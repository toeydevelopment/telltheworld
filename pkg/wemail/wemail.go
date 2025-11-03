package wemail

import (
	"context"
	"errors"
	"html/template"

	emailverifier "github.com/AfterShip/email-verifier"
	"github.com/wneessen/go-mail"
)

var (
	ErrDisposableEmail  = errors.New("disposable email")
	ErrUnreachableEmail = errors.New("unreachable email")
	ErrInvalidEmail     = errors.New("invalid email")
)

type Validator struct {
	verifier *emailverifier.Verifier
}

type Sender struct {
	client *mail.Client
}

type SenderConfig struct {
	Host     string
	Port     int
	Email    string
	Password string
}

func NewValidator() *Validator {
	return &Validator{
		verifier: emailverifier.NewVerifier(),
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	email string,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	result, err := v.verifier.Verify(email)
	if err != nil {
		return err
	}

	if !result.Syntax.Valid {
		return ErrInvalidEmail
	}

	if result.Disposable {
		return ErrDisposableEmail
	}

	if result.Reachable == "no" {
		return ErrUnreachableEmail
	}

	return nil
}

func NewSender(cfg SenderConfig) (*Sender, error) {
	client, err := mail.NewClient(
		cfg.Host,
		mail.WithPort(cfg.Port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(cfg.Email),
		mail.WithPassword(cfg.Password),
	)

	if err != nil {
		return nil, err
	}

	return &Sender{client: client}, nil
}

type Message struct {
	From    string
	To      []string
	Subject string
	// Body is the body of the email in HTML format.
	Body *template.Template
	Data any
}

func (s *Sender) Send(ctx context.Context, msg Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	message := mail.NewMsg()
	if err := message.From(msg.From); err != nil {
		return err
	}
	if err := message.To(msg.To...); err != nil {
		return err
	}
	message.Subject(msg.Subject)

	if err := message.SetBodyHTMLTemplate(msg.Body, msg.Data); err != nil {
		return err
	}

	return s.client.DialAndSendWithContext(ctx, message)
}
