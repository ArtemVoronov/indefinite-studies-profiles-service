package templates

import "github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/kafka"

// TODO: add using freemarker
type EmailTemplateService struct {
	baseUrl         string
	senderEmailAddr string
}

func NewEmailTemplateService(baseUrl string, senderEmailAddr string) *EmailTemplateService {
	return &EmailTemplateService{
		baseUrl:         baseUrl,
		senderEmailAddr: senderEmailAddr,
	}
}

func (s *EmailTemplateService) Shutdown() error {
	return nil
}

func (s *EmailTemplateService) GetEmailSignUpConfirmationLink(email string, token string) kafka.SendEmailEvent {
	link := s.baseUrl + "/signup/" + token

	return kafka.SendEmailEvent{
		Sender:    s.senderEmailAddr,
		Recepient: email,
		Subject:   "Registration at indefinitestudies.ru",
		Body:      "Welcome!\n\nUse the following link for finishing registration: " + link + "\n\nBest Regards,\nIndefinite Studies Team",
	}
}

func (s *EmailTemplateService) GetEmailRestorePasswordLink(email string, token string) kafka.SendEmailEvent {
	link := s.baseUrl + "/restorepwd/" + token

	return kafka.SendEmailEvent{
		Sender:    s.senderEmailAddr,
		Recepient: email,
		Subject:   "Restore password",
		Body:      "Hello!\n\nUse the following link for restoring password: " + link + "\n\nBest Regards,\nIndefinite Studies Team",
	}
}
