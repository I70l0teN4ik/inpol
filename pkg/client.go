package pkg

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const refresh = "/identity/refresh"

type Client interface {
	Login(username string, password string) (string, error)
	RefreshToken() error
	Dates(ctx context.Context) ([]string, error)
	Slots(ctx context.Context, date string) ([]Slot, error)
	Reserve(ctx context.Context, slot Slot) (bool, error)
	GetMFA(ctx context.Context) string
	NotifyTelegramUsers(message string) error
}

type client struct {
	JWT    string
	MFA    string
	conf   Config
	logger *log.Logger
	client *http.Client
}

func (c *client) NotifyTelegramUsers(message string) error {
	botToken := c.conf.TelegramBotToken
	chatIDs := c.conf.TelegramChatIDs

	// Skip if not configured
	if botToken == "" || len(chatIDs) == 0 {
		c.logger.Println("Telegram notification skipped: missing configuration")
		return nil
	}

	err := SendTelegramMessage(botToken, chatIDs, message)
	if err != nil {
		c.logger.Printf("Failed to send Telegram notification: %v", err)
		return err
	}

	c.logger.Println("Telegram notification sent successfully")
	return nil
}

func NewClient(conf Config, jwt string) (Client, error) {
	cl := http.DefaultClient
	jar, _ := cookiejar.New(nil)

	cl.Jar = jar
	//cl.Timeout = time.Second * 10

	c := &client{
		conf:   conf,
		JWT:    jwt,
		MFA:    conf.MFA,
		logger: log.New(os.Stdout, "cln: ", log.Ltime),
		client: cl,
	}

	return c, c.validateToken()
}

func (c *client) Login(username string, password string) (string, error) {

	//req, err := c.prepareRequest(context.Background(), &url.URL{Scheme: "https", Host: c.conf.Host, Path: refresh}, nil)
	panic("implement me")
}

func (c *client) RefreshToken() error {
	c.logger.Println("Refreshing token...")

	req, err := c.prepareRequest(context.Background(), &url.URL{Scheme: "https", Host: c.conf.Host, Path: refresh}, nil)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	c.logger.Printf("Refreshed status: %d\n", res.StatusCode)

	if res.StatusCode == http.StatusOK {
		if responseData, err := io.ReadAll(res.Body); err == nil {
			c.JWT = string(responseData)
			c.logger.Println("new token", c.JWT)
		} else {
			return err
		}
	}

	return nil
}

func (c *client) Dates(ctx context.Context) ([]string, error) {
	req, err := c.prepareRequest(ctx, c.reserveUrl("dates"), http.NoBody)
	if err != nil {
		return nil, err
	}
	c.logger.Println("Requesting dates")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("status code: %d", res.StatusCode))
	}

	var dates []string
	err = json.NewDecoder(res.Body).Decode(&dates)
	if err != nil {
		return nil, err
	}

	c.logger.Println("dates", dates)

	return dates, nil
}

func (c *client) Reserve(ctx context.Context, slot Slot) (bool, error) {
	reqBody := Reserve{
		SlotId:       slot.Id,
		ProceedingId: c.conf.Case,
		Name:         c.conf.Name,
		LastName:     c.conf.LastName,
		DateOfBirth:  c.conf.DateOfBirth,
	}
	c.logger.Println(reqBody)
	enc, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := c.prepareRequest(ctx, c.reserveUrl("reserve"), bytes.NewReader(enc))
	if err != nil {
		return false, err
	}

	// use pre generated MFA from .env
	mfa := c.conf.MFA

	if mfa == "" {
		mfa = c.GetMFA(ctx)
	}

	req.Header.Set("2fa", "Bearer "+mfa)

	res, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	c.logger.Println("reserve status", res.StatusCode)

	responseData, err := io.ReadAll(res.Body)
	c.logger.Println("reserve str", string(responseData))

	return res.StatusCode == http.StatusOK, err
}

func (c *client) GetMFA(ctx context.Context) string {
	mfaURL := &url.URL{Scheme: "https", Host: c.conf.Host, Path: "/identity/two-factor"}
	verifyURL := &url.URL{Scheme: "https", Host: c.conf.Host, Path: "/identity/two-factor-verification"}

	enc, err := json.Marshal(map[string]string{"purpose": "MakeAppointment"})
	clickReq, err := c.prepareRequest(ctx, mfaURL, bytes.NewReader(enc))
	if err != nil {
		c.logger.Println(err)
		return ""
	}

	res, err := c.client.Do(clickReq)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		c.logger.Println("MFA status", res.StatusCode)
		return ""
	}

	var data map[string]string
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		c.logger.Println(err)
		return ""
	}

	data["verificationCode"] = c.readCode()
	delete(data, "provider")

	mfaBody, err := json.Marshal(data)
	mfaReq, err := c.prepareRequest(ctx, verifyURL, bytes.NewReader(mfaBody))
	if err != nil {
		c.logger.Println(err)
		return ""
	}

	mfaRes, err := c.client.Do(mfaReq)
	if err != nil {
		return ""
	}
	defer mfaRes.Body.Close()
	if mfaRes.StatusCode != http.StatusOK {
		c.logger.Println("MFA verify status", mfaRes.StatusCode)
		return ""
	}

	var mfaData map[string]string
	err = json.NewDecoder(mfaRes.Body).Decode(&mfaData)
	if err != nil {
		c.logger.Println(err)
		return ""
	}

	return mfaData["confirmedToken"]
}

func (c *client) Slots(ctx context.Context, date string) ([]Slot, error) {
	var slots []Slot

	req, err := c.prepareRequest(ctx, c.reserveUrl(date+"/slots"), http.NoBody)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&slots)

	c.logger.Println("slots for "+date, res.StatusCode, slots)

	return slots, nil
}

func (c *client) reserveUrl(path string) *url.URL {
	return &url.URL{Scheme: "https", Host: c.conf.Host, Path: fmt.Sprintf("/api/reservations/queue/%s/%s", c.conf.Queue, path)}
}

func (c *client) prepareRequest(ctx context.Context, endpoint *url.URL, body io.Reader) (*http.Request, error) {
	if endpoint.Path != refresh {
		err := c.validateToken()
		if err != nil {
			c.logger.Println(err)
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		body,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.JWT)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://inpol.mazowieckie.pl/home/cases/"+c.conf.Case)
	req.Header.Set("Origin", "https://inpol.mazowieckie.pl")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")

	req.AddCookie(&http.Cookie{Name: "cookieconsent_status", Value: "dismiss"})
	req.AddCookie(&http.Cookie{Name: "visid_incap_3087338", Value: "6XZCquQpQcKnEDs1a5ARGUXGwmcAAAAAQUIPAAAAAAALdX1exslSdO/ZocZg5Oqn"})

	return req, nil
}

func (c *client) validateToken() error {
	if c.JWT == "" {
		return fmt.Errorf("JWT token is empty")
	}

	claims := make(jwt.MapClaims)
	token, _, err := jwt.NewParser().ParseUnverified(c.JWT, claims)

	if err != nil {
		return fmt.Errorf("failed to parse token: %s", err)
	}

	exp, _ := token.Claims.GetExpirationTime()
	//fmt.Println("exp", exp.Format("2006-01-02 15:04:05"))

	if exp.Before(time.Now()) {
		panic("token expired")
		return errors.New("token expired")
	}

	if time.Now().After(exp.Add(time.Minute * -2)) {
		c.logger.Println("Time to refresh...")

		if err = c.RefreshToken(); err != nil {
			c.logger.Println("Failed to refresh token: ", err)

			return errors.New("failed to refresh token")
		}
	}

	return nil
}

func (c *client) readCode() string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter Code: ")
	scanner.Scan()
	return scanner.Text()
}
