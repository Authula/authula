package auth

import "github.com/GoBetterAuth/go-better-auth/models"

type MeResult struct {
	User    *models.User    `json:"user"`
	Session *models.Session `json:"session"`
}

// GetMe retrieves the current user and their session
func (s *Service) GetMe(userID string) (*MeResult, error) {
	user, err := s.UserService.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	session, err := s.SessionService.GetSessionByUserID(userID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	return &MeResult{
		User:    user,
		Session: session,
	}, nil
}
