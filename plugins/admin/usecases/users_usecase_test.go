package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/uptrace/bun"

	corerepositories "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type mockUserRepositoryAdapter struct {
	mock *internaltests.MockUserRepository
}

func newUsersUseCase(mockRepo *internaltests.MockUserRepository) usecases.UsersUseCase {
	return usecases.NewUsersUseCase(&mockUserRepositoryAdapter{mock: mockRepo})
}

func (a *mockUserRepositoryAdapter) GetAll(ctx context.Context, cursor *string, limit int) ([]models.User, *string, error) {
	return a.mock.GetAll(ctx, cursor, limit)
}

func (a *mockUserRepositoryAdapter) GetByID(ctx context.Context, id string) (*models.User, error) {
	return a.mock.GetByID(ctx, id)
}

func (a *mockUserRepositoryAdapter) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return a.mock.GetByEmail(ctx, email)
}

func (a *mockUserRepositoryAdapter) Create(ctx context.Context, user *models.User) (*models.User, error) {
	return a.mock.Create(ctx, user.Name, user.Email, user.EmailVerified, user.Image, user.Metadata)
}

func (a *mockUserRepositoryAdapter) Update(ctx context.Context, user *models.User) (*models.User, error) {
	return a.mock.Update(ctx, user)
}

func (a *mockUserRepositoryAdapter) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	return a.mock.UpdateFields(ctx, id, fields)
}

func (a *mockUserRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return a.mock.Delete(ctx, id)
}

func (a *mockUserRepositoryAdapter) WithTx(tx bun.IDB) corerepositories.UserRepository {
	return a
}

// ============================================================================
// GetAll Tests
// ============================================================================

func TestUsersUseCase_GetAll_CursorPagination(t *testing.T) {
	mockRepo := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockRepo)

	// Mock first page (2 users + cursor)
	usersGroupOne := []models.User{{ID: "u-001"}, {ID: "u-002"}}
	cursor := "abc123"
	mockRepo.On("GetAll", mock.Anything, mock.Anything, 2).Return(usersGroupOne, &cursor, nil).Once()

	// Mock second page (1 user)
	usersGroupTwo := []models.User{{ID: "u-003"}}
	mockRepo.On("GetAll", mock.Anything, mock.AnythingOfType("*string"), 2).Return(usersGroupTwo, nil, nil).Once()

	// Test first page
	page1, err := uc.GetAll(context.Background(), nil, 2)
	assert.NoError(t, err)
	assert.Len(t, page1.Users, 2)
	assert.Equal(t, "abc123", *page1.NextCursor)

	// Test second page
	page2, err := uc.GetAll(context.Background(), page1.NextCursor, 2)
	assert.NoError(t, err)
	assert.Len(t, page2.Users, 1)

	mockRepo.AssertExpectations(t)
}

func TestUsersUseCase_GetAll_DefaultLimit(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	users := []models.User{{ID: "u-001"}}
	mockSvc.On("GetAll", mock.Anything, mock.Anything, 20).Return(users, nil, nil).Once()

	page, err := uc.GetAll(context.Background(), nil, 0)
	assert.NoError(t, err)
	assert.Len(t, page.Users, 1)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_GetAll_LimitCapped(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	users := []models.User{{ID: "u-001"}}
	mockSvc.On("GetAll", mock.Anything, mock.Anything, 100).Return(users, nil, nil).Once()

	page, err := uc.GetAll(context.Background(), nil, 500)
	assert.NoError(t, err)
	assert.Len(t, page.Users, 1)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_GetAll_CursorTrimmed(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	users := []models.User{{ID: "u-001"}}
	mockSvc.On("GetAll", mock.Anything, mock.MatchedBy(func(c *string) bool {
		return c != nil && *c == "trimmed"
	}), 20).Return(users, nil, nil).Once()

	cursorWithSpaces := "  trimmed  "
	page, err := uc.GetAll(context.Background(), &cursorWithSpaces, 20)
	assert.NoError(t, err)
	assert.Len(t, page.Users, 1)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_GetAll_ServiceError(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	serviceErr := errors.New("database error")
	mockSvc.On("GetAll", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil, serviceErr).Once()

	page, err := uc.GetAll(context.Background(), nil, 20)
	assert.Error(t, err)
	assert.Nil(t, page)
	assert.Equal(t, serviceErr, err)

	mockSvc.AssertExpectations(t)
}

// ============================================================================
// GetByID Tests
// ============================================================================

func TestUsersUseCase_GetByID_Success(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	user := &models.User{ID: "u-001", Name: "John", Email: "john@example.com"}
	mockSvc.On("GetByID", mock.Anything, "u-001").Return(user, nil).Once()

	result, err := uc.GetByID(context.Background(), "u-001")
	assert.NoError(t, err)
	assert.Equal(t, user, result)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_GetByID_EmptyID(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	result, err := uc.GetByID(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "user_id is required", err.Error())

	mockSvc.AssertNotCalled(t, "GetByID")
}

func TestUsersUseCase_GetByID_WhitespaceOnlyID(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	result, err := uc.GetByID(context.Background(), "   ")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "user_id is required", err.Error())

	mockSvc.AssertNotCalled(t, "GetByID")
}

func TestUsersUseCase_GetByID_ServiceError(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	serviceErr := errors.New("database error")
	mockSvc.On("GetByID", mock.Anything, "u-001").Return(nil, serviceErr).Once()

	result, err := uc.GetByID(context.Background(), "u-001")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, serviceErr, err)

	mockSvc.AssertExpectations(t)
}

// ============================================================================
// Create Tests
// ============================================================================

func TestUsersUseCase_Create_Success(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	req := types.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	createdUser := &models.User{ID: "u-001", Name: "John Doe", Email: "john@example.com", EmailVerified: false}
	mockSvc.On("GetByEmail", mock.Anything, "john@example.com").Return(nil, nil).Once()
	mockSvc.On("Create", mock.Anything, "John Doe", "john@example.com", false, (*string)(nil), json.RawMessage(nil)).Return(createdUser, nil).Once()

	result, err := uc.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, createdUser, result)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_Create_EmailVerifiedTrue(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	emailVerified := true
	req := types.CreateUserRequest{
		Name:          "Jane Doe",
		Email:         "jane@example.com",
		EmailVerified: &emailVerified,
	}

	createdUser := &models.User{ID: "u-002", Name: "Jane Doe", Email: "jane@example.com", EmailVerified: true}
	mockSvc.On("GetByEmail", mock.Anything, "jane@example.com").Return(nil, nil).Once()
	mockSvc.On("Create", mock.Anything, "Jane Doe", "jane@example.com", true, (*string)(nil), json.RawMessage(nil)).Return(createdUser, nil).Once()

	result, err := uc.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, createdUser, result)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_Create_WithImage(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	imageURL := "https://example.com/image.jpg"
	req := types.CreateUserRequest{
		Name:  "John",
		Email: "john@example.com",
		Image: &imageURL,
	}

	createdUser := &models.User{ID: "u-003", Name: "John", Email: "john@example.com", Image: &imageURL}
	mockSvc.On("GetByEmail", mock.Anything, "john@example.com").Return(nil, nil).Once()
	mockSvc.On("Create", mock.Anything, "John", "john@example.com", false, &imageURL, json.RawMessage(nil)).Return(createdUser, nil).Once()

	result, err := uc.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, createdUser, result)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_Create_EmptyName(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	req := types.CreateUserRequest{
		Name:  "",
		Email: "john@example.com",
	}

	result, err := uc.Create(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "name is required", err.Error())

	mockSvc.AssertNotCalled(t, "GetByEmail")
}

func TestUsersUseCase_Create_EmptyEmail(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	req := types.CreateUserRequest{
		Name:  "John Doe",
		Email: "",
	}

	result, err := uc.Create(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "email is required", err.Error())

	mockSvc.AssertNotCalled(t, "GetByEmail")
}

func TestUsersUseCase_Create_UserAlreadyExists(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	req := types.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	existingUser := &models.User{ID: "u-existing", Name: "John Doe", Email: "john@example.com"}
	mockSvc.On("GetByEmail", mock.Anything, "john@example.com").Return(existingUser, nil).Once()

	result, err := uc.Create(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "user already exists", err.Error())

	mockSvc.AssertNotCalled(t, "Create")
}

func TestUsersUseCase_Create_GetByEmailServiceError(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	req := types.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	serviceErr := errors.New("database error")
	mockSvc.On("GetByEmail", mock.Anything, "john@example.com").Return(nil, serviceErr).Once()

	result, err := uc.Create(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, serviceErr, err)

	mockSvc.AssertNotCalled(t, "Create")
}

func TestUsersUseCase_Create_ServiceError(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	req := types.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	mockSvc.On("GetByEmail", mock.Anything, "john@example.com").Return(nil, nil).Once()
	serviceErr := errors.New("create failed")
	mockSvc.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, serviceErr).Once()

	result, err := uc.Create(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, serviceErr, err)

	mockSvc.AssertExpectations(t)
}

// ============================================================================
// Update Tests
// ============================================================================

func TestUsersUseCase_Update_UpdateName(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	existingUser := &models.User{ID: "u-001", Name: "Old Name", Email: "john@example.com"}
	newName := "New Name"
	req := types.UpdateUserRequest{Name: &newName}

	mockSvc.On("GetByID", mock.Anything, "u-001").Return(existingUser, nil).Once()
	updatedUser := &models.User{ID: "u-001", Name: "New Name", Email: "john@example.com"}
	mockSvc.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.ID == "u-001" && u.Name == "New Name"
	})).Return(updatedUser, nil).Once()

	result, err := uc.Update(context.Background(), "u-001", req)
	assert.NoError(t, err)
	assert.Equal(t, updatedUser, result)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_Update_UpdateEmail(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	existingUser := &models.User{ID: "u-001", Name: "John", Email: "old@example.com"}
	newEmail := "new@example.com"
	req := types.UpdateUserRequest{Email: &newEmail}

	mockSvc.On("GetByID", mock.Anything, "u-001").Return(existingUser, nil).Once()
	updatedUser := &models.User{ID: "u-001", Name: "John", Email: "new@example.com"}
	mockSvc.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.ID == "u-001" && u.Email == "new@example.com"
	})).Return(updatedUser, nil).Once()

	result, err := uc.Update(context.Background(), "u-001", req)
	assert.NoError(t, err)
	assert.Equal(t, updatedUser, result)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_Update_UpdateEmailVerified(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	existingUser := &models.User{ID: "u-001", Name: "John", Email: "john@example.com", EmailVerified: false}
	emailVerified := true
	req := types.UpdateUserRequest{EmailVerified: &emailVerified}

	mockSvc.On("GetByID", mock.Anything, "u-001").Return(existingUser, nil).Once()
	updatedUser := &models.User{ID: "u-001", Name: "John", Email: "john@example.com", EmailVerified: true}
	mockSvc.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.ID == "u-001" && u.EmailVerified
	})).Return(updatedUser, nil).Once()

	result, err := uc.Update(context.Background(), "u-001", req)
	assert.NoError(t, err)
	assert.Equal(t, updatedUser, result)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_Update_UpdateImage(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	existingUser := &models.User{ID: "u-001", Name: "John", Email: "john@example.com"}
	newImage := "https://example.com/new-image.jpg"
	req := types.UpdateUserRequest{Image: &newImage}

	mockSvc.On("GetByID", mock.Anything, "u-001").Return(existingUser, nil).Once()
	updatedUser := &models.User{ID: "u-001", Name: "John", Email: "john@example.com", Image: &newImage}
	mockSvc.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.ID == "u-001" && u.Image != nil && *u.Image == newImage
	})).Return(updatedUser, nil).Once()

	result, err := uc.Update(context.Background(), "u-001", req)
	assert.NoError(t, err)
	assert.Equal(t, updatedUser, result)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_Update_EmptyUserID(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	newName := "New Name"
	req := types.UpdateUserRequest{Name: &newName}

	result, err := uc.Update(context.Background(), "", req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "user_id is required", err.Error())

	mockSvc.AssertNotCalled(t, "GetByID")
}

func TestUsersUseCase_Update_NoFieldsProvided(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	req := types.UpdateUserRequest{}

	result, err := uc.Update(context.Background(), "u-001", req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "at least one field is required for update", err.Error())

	mockSvc.AssertNotCalled(t, "GetByID")
}

func TestUsersUseCase_Update_UserNotFound(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	newName := "New Name"
	req := types.UpdateUserRequest{Name: &newName}

	mockSvc.On("GetByID", mock.Anything, "u-001").Return(nil, nil).Once()

	result, err := uc.Update(context.Background(), "u-001", req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "user not found", err.Error())

	mockSvc.AssertNotCalled(t, "Update")
}

func TestUsersUseCase_Update_GetByIDServiceError(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	newName := "New Name"
	req := types.UpdateUserRequest{Name: &newName}

	serviceErr := errors.New("database error")
	mockSvc.On("GetByID", mock.Anything, "u-001").Return(nil, serviceErr).Once()

	result, err := uc.Update(context.Background(), "u-001", req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, serviceErr, err)

	mockSvc.AssertNotCalled(t, "Update")
}

func TestUsersUseCase_Update_ServiceError(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	existingUser := &models.User{ID: "u-001", Name: "Old Name", Email: "john@example.com"}
	newName := "New Name"
	req := types.UpdateUserRequest{Name: &newName}

	mockSvc.On("GetByID", mock.Anything, "u-001").Return(existingUser, nil).Once()
	serviceErr := errors.New("update failed")
	mockSvc.On("Update", mock.Anything, mock.Anything).Return(nil, serviceErr).Once()

	result, err := uc.Update(context.Background(), "u-001", req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, serviceErr, err)

	mockSvc.AssertExpectations(t)
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestUsersUseCase_Delete_Success(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	existingUser := &models.User{ID: "u-001", Name: "John", Email: "john@example.com"}
	mockSvc.On("GetByID", mock.Anything, "u-001").Return(existingUser, nil).Once()
	mockSvc.On("Delete", mock.Anything, "u-001").Return(nil).Once()

	err := uc.Delete(context.Background(), "u-001")
	assert.NoError(t, err)

	mockSvc.AssertExpectations(t)
}

func TestUsersUseCase_Delete_EmptyUserID(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	err := uc.Delete(context.Background(), "")
	assert.Error(t, err)
	assert.Equal(t, "user_id is required", err.Error())

	mockSvc.AssertNotCalled(t, "GetByID")
}

func TestUsersUseCase_Delete_UserNotFound(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	mockSvc.On("GetByID", mock.Anything, "u-001").Return(nil, nil).Once()

	err := uc.Delete(context.Background(), "u-001")
	assert.Error(t, err)
	assert.Equal(t, "user not found", err.Error())

	mockSvc.AssertNotCalled(t, "Delete")
}

func TestUsersUseCase_Delete_ServiceError(t *testing.T) {
	mockSvc := new(internaltests.MockUserRepository)
	uc := newUsersUseCase(mockSvc)

	existingUser := &models.User{ID: "u-001", Name: "John", Email: "john@example.com"}
	mockSvc.On("GetByID", mock.Anything, "u-001").Return(existingUser, nil).Once()
	serviceErr := errors.New("delete failed")
	mockSvc.On("Delete", mock.Anything, "u-001").Return(serviceErr).Once()

	err := uc.Delete(context.Background(), "u-001")
	assert.Error(t, err)
	assert.Equal(t, serviceErr, err)

	mockSvc.AssertExpectations(t)
}
