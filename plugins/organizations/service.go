package organizations

import "context"

type OrganizationLookupService interface {
	ExistsByID(ctx context.Context, organizationID string) (bool, error)
}
