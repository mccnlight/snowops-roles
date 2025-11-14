package models

// Роли пользователей.
const (
	RoleAkimatAdmin     = "AKIMAT_ADMIN"
	RoleKguAdmin        = "KGU_ADMIN"
	RoleContractorAdmin = "CONTRACTOR_ADMIN"
	RoleDriver          = "DRIVER"
)

// Типы организаций.
const (
	OrgTypeAkimat     = "AKIMAT"
	OrgTypeKgu        = "KGU"
	OrgTypeContractor = "CONTRACTOR"
)

// IsAdmin проверяет, относится ли роль к административным.
func IsAdmin(role string) bool {
	switch role {
	case RoleAkimatAdmin, RoleKguAdmin, RoleContractorAdmin:
		return true
	default:
		return false
	}
}

// CanCreateOrganization определяет, может ли роль создавать организации заданного типа.
func CanCreateOrganization(role, orgType string) bool {
	switch role {
	case RoleAkimatAdmin:
		return orgType == OrgTypeKgu
	case RoleKguAdmin:
		return orgType == OrgTypeContractor
	default:
		return false
	}
}

// IsAkimatAdmin проверяет, является ли роль администратором акимата.
func IsAkimatAdmin(role string) bool {
	return role == RoleAkimatAdmin
}

// IsKguAdmin проверяет, является ли роль администратором КГУ.
func IsKguAdmin(role string) bool {
	return role == RoleKguAdmin
}

// IsContractorAdmin проверяет, является ли роль администратором подрядчика.
func IsContractorAdmin(role string) bool {
	return role == RoleContractorAdmin
}

// IsDriver проверяет, является ли роль водителем.
func IsDriver(role string) bool {
	return role == RoleDriver
}
