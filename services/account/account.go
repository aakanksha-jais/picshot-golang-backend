package account

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/auth"

	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/app"

	"github.com/Aakanksha-jais/picshot-golang-backend/models"
	"github.com/Aakanksha-jais/picshot-golang-backend/pkg/errors"
	"github.com/Aakanksha-jais/picshot-golang-backend/services"
	"github.com/Aakanksha-jais/picshot-golang-backend/stores"
	"golang.org/x/crypto/bcrypt"
)

type account struct {
	accountStore stores.Account
	blogService  services.Blog
}

func (a account) SendOTP(ctx *app.Context, phone string) (*models.VerificationResponse, error) {
	err := validatePhone(phone)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("FriendlyName", "PicShot (phone number)")
	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest(http.MethodPost, "https://verify.twilio.com/v2/Services", body)
	if err != nil {
		return nil, errors.Error{Err: err}
	}

	req.SetBasicAuth(ctx.Get("TWILIO_ACCOUNT_SID"), ctx.Get("TWILIO_AUTH_TOKEN")) // set auth header
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Error{Err: err}
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.BodyRead{Err: err, Msg: "cannot read response body of twilio.com"}
	}

	type links struct {
		Verifications      string `json:"verifications"`
		VerificationChecks string `json:"verification_checks"`
	}
	resp := struct {
		Links links `json:"links"`
	}{}

	err = json.Unmarshal(bodyBytes, &resp)
	if err != nil {
		return nil, errors.Unmarshal{Err: err, Msg: "cannot unmarshal response from twilio.com"}
	}

	params = url.Values{}
	params.Set("To", phone)
	params.Set("Channel", "sms")

	body = strings.NewReader(params.Encode())

	req, err = http.NewRequest(http.MethodPost, resp.Links.Verifications, body)
	if err != nil {
		return nil, errors.Error{Err: err}
	}

	req.SetBasicAuth(ctx.Get("TWILIO_ACCOUNT_SID"), ctx.Get("TWILIO_AUTH_TOKEN"))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Error{Err: err}
	}

	bodyBytes, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.BodyRead{Err: err, Msg: "cannot read response body of twilio.com"}
	}

	var sid struct {
		SID string `json:"sid"`
	}

	err = json.Unmarshal(bodyBytes, &sid)
	if err != nil {
		return nil, errors.Unmarshal{Err: err, Msg: "cannot unmarshal response from twilio.com"}
	}

	return &models.VerificationResponse{URL: resp.Links.VerificationChecks, SID: sid.SID}, nil
}
func (a account) VerifyPhone(ctx *app.Context, sid, otp, reqURL string) error {
	params := url.Values{}
	params.Set("Code", otp)
	params.Set("VerificationSid", sid)

	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest(http.MethodPost, reqURL, body)
	if err != nil {
		return errors.Error{Err: err}
	}

	req.SetBasicAuth(ctx.Get("TWILIO_ACCOUNT_SID"), ctx.Get("TWILIO_AUTH_TOKEN"))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Error{Err: err}
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.BodyRead{Err: err, Msg: "cannot read response body of twilio.com"}
	}

	var resp struct {
		Status interface{} `json:"status"`
	}

	err = json.Unmarshal(bodyBytes, &resp)
	if err != nil {
		return errors.Unmarshal{Err: err, Msg: "cannot unmarshal response from twilio.com"}
	}

	status, ok := resp.Status.(string)
	if ok && status == "approved" {
		return nil
	}

	return errors.InvalidParam{Param: "OTP"}
}

func New(accountStore stores.Account, blogService services.Blog) services.Account {
	return account{
		accountStore: accountStore,
		blogService:  blogService,
	}
}

// GetAll gets all accounts that match the filter.
func (a account) GetAll(ctx *app.Context, filter *models.Account) ([]*models.Account, error) {
	return a.accountStore.GetAll(ctx, filter)
}

func (a account) GetByID(ctx *app.Context, id int64) (*models.Account, error) {
	account, err := a.accountStore.Get(ctx, &models.Account{User: models.User{ID: id}})
	if err != nil {
		return nil, err
	}

	if account == nil || reflect.DeepEqual(account, &models.Account{}) {
		return nil, errors.EntityNotFound{Entity: "user"}
	}

	account.Password = ""

	return account, nil
}

// GetAccountWithBlogs fetches an account with all the blogs posted by the account.
func (a account) GetAccountWithBlogs(ctx *app.Context, username string) (*models.Account, error) {
	err := validateUsername(username)
	if err != nil {
		return nil, err
	}

	account, err := a.accountStore.Get(ctx, &models.Account{User: models.User{UserName: username}})
	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, errors.EntityNotFound{Entity: "user"}
	}

	blogs, err := a.blogService.GetAll(ctx, &models.Blog{AccountID: account.ID}, nil)
	if err != nil {
		return nil, err
	}

	for i := range blogs {
		if blogs[i] != nil {
			account.Blogs = append(account.Blogs, *blogs[i])
		}
	}

	return account, nil
}

func (a account) UpdateUser(ctx *app.Context, user *models.User) (*models.Account, error) {
	if user == nil {
		return nil, errors.MissingParam{Param: "user details"}
	}

	jwtIDKey := auth.JWTContextKey("claims")
	id := ctx.Value(jwtIDKey).(*auth.Claims).UserID

	account, err := a.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	update, err := a.getUpdate(ctx, account, user)
	if err != nil {
		return nil, err
	}

	update.ID = id

	return a.accountStore.Update(ctx, update)
}

// Update updates account information based on account_id.todo
func (a account) Update(ctx *app.Context, model *models.Account, id int64) (*models.Account, error) {
	model.ID = id

	return a.accountStore.Update(ctx, model)
}

func (a account) UpdatePassword(ctx *app.Context, oldPassword, newPassword string) error {
	jwtIDKey := auth.JWTContextKey("claims")
	id := ctx.Value(jwtIDKey).(*auth.Claims).UserID

	account, err := a.accountStore.Get(ctx, &models.Account{User: models.User{ID: id}})
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(oldPassword))
	if err != nil {
		return errors.AuthError{Err: err, Msg: "invalid password"}
	}

	err = validatePassword(newPassword)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Error{Err: err, Msg: "error in hashing password", Type: "password-hash-error"}
	}

	_, err = a.accountStore.Update(ctx, &models.Account{
		User:      models.User{ID: id, Password: string(hash)},
		PwdUpdate: sql.NullTime{Time: time.Now(), Valid: true},
	})

	return err
}

// Delete deactivates an account and updates it's deletion request.
// After 30 days, the account gets deleted if the status remains inactive.
func (a account) Delete(ctx *app.Context) error { // TODO: trigger a cronjob for 30 days deletion functionality
	jwtIDKey := auth.JWTContextKey("claims")
	id := ctx.Value(jwtIDKey).(*auth.Claims).UserID

	return a.accountStore.Delete(ctx, id)
}

// Create creates an account and assigns an id to it.
func (a account) Create(ctx *app.Context, user *models.User) (*models.Account, error) {
	if user == nil {
		return nil, errors.MissingParam{Param: "user details"}
	}

	// check that the user does not exist already
	err := a.checkUserExists(ctx, user)
	if err != nil {
		return nil, err
	}

	// check if user details are valid
	err = validateUser(user, ctx.Config)
	if err != nil {
		return nil, err
	}

	account := models.Account{User: *user, Status: "ACTIVE"}

	password, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Error{Err: err, Msg: "error in hashing password", Type: "password-hash-error"}
	}

	account.Password = string(password)

	return a.accountStore.Create(ctx, &account)
}

// CheckAvailability checks if user name exists in the database.
func (a account) CheckAvailability(ctx *app.Context, user *models.User) error {
	if empty(user) {
		return errors.MissingParam{Param: "signup_id"}
	}

	if user.UserName == "" {
		if user.Email.String == "" {
			if err := validatePhone(user.PhoneNo.String); err != nil {
				return err
			}

			return a.checkPhoneAvailability(ctx, user.PhoneNo.String)
		}

		if err := validateEmail(user.Email.String); err != nil {
			return err
		}

		if err := realEmail(user.Email.String, ctx.Get("REALMAIL_API_KEY")); err != nil {
			return err
		}

		return a.checkEmailAvailability(ctx, user.Email.String)
	}

	if err := validateUsername(user.UserName); err != nil {
		return err
	}

	return a.checkUsernameAvailability(ctx, user.UserName)
}

// Login gets an account by the User Details filter.
func (a account) Login(ctx *app.Context, user *models.User) (*models.Account, error) {
	if user == nil {
		return nil, errors.MissingParam{Param: "user details"}
	}

	if empty(user) {
		return nil, errors.MissingParam{Param: "login_id"}
	}

	if user.UserName != "" {
		err := validateUsername(user.UserName)
		if err != nil {
			return nil, err
		}
	}

	if user.Email.String != "" {
		if err := validateEmail(user.Email.String); err != nil {
			return nil, err
		}
	}

	if user.PhoneNo.String != "" {
		if err := validatePhone(user.PhoneNo.String); err != nil {
			return nil, err
		}
	}

	account, err := a.accountStore.Get(ctx, &models.Account{User: *user})
	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, errors.EntityNotFound{Entity: "user"}
	}

	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(user.Password))
	if err != nil {
		return nil, errors.AuthError{Err: err, Msg: "invalid password"}
	}

	return a.Update(ctx, &models.Account{DelRequest: sql.NullTime{}, Status: "ACTIVE"}, account.ID)
}
