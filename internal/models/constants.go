package models

// Роли пользователей.
const (
	RoleAkimatAdmin     = "AKIMAT_ADMIN"
	RoleAkimatUser      = "AKIMAT_USER"
	RoleKguZkhAdmin     = "KGU_ZKH_ADMIN"
	RoleKguZkhUser      = "KGU_ZKH_USER"
	RoleLandfillAdmin   = "LANDFILL_ADMIN"
	RoleLandfillUser    = "LANDFILL_USER"
	RoleContractorAdmin = "CONTRACTOR_ADMIN"
	RoleContractorUser  = "CONTRACTOR_USER"
	RoleDriver          = "DRIVER"
)

// Типы организаций.
const (
	OrgTypeAkimat     = "AKIMAT"
	OrgTypeKguZkh     = "KGU_ZKH"
	OrgTypeLandfill   = "LANDFILL"
	OrgTypeToo        = "TOO"
	OrgTypeContractor = "CONTRACTOR"
)

// IsAdmin проверяет, относится ли роль к административным.
func IsAdmin(role string) bool {
	switch role {
	case RoleAkimatAdmin, RoleKguZkhAdmin, RoleLandfillAdmin, RoleContractorAdmin:
		return true
	default:
		return false
	}
}

// CanCreateOrganization определяет, может ли роль создавать организации заданного типа.
func CanCreateOrganization(role, orgType string) bool {
	switch role {
	case RoleAkimatAdmin, RoleAkimatUser:
		return orgType == OrgTypeKguZkh || orgType == OrgTypeLandfill || orgType == OrgTypeToo
	case RoleKguZkhAdmin, RoleKguZkhUser:
		return orgType == OrgTypeContractor || orgType == OrgTypeLandfill || orgType == OrgTypeToo
	default:
		return false
	}
}

// IsAkimatAdmin проверяет, является ли роль администратором акимата.
func IsAkimatAdmin(role string) bool {
	return role == RoleAkimatAdmin
}

// IsAkimatRole проверяет, является ли роль пользователем акимата (admin или user).
func IsAkimatRole(role string) bool {
	return role == RoleAkimatAdmin || role == RoleAkimatUser
}

// IsKguAdmin проверяет, является ли роль администратором КГУ.
func IsKguAdmin(role string) bool {
	return role == RoleKguZkhAdmin
}

// IsKguRole проверяет, является ли роль пользователем КГУ (admin или user).
func IsKguRole(role string) bool {
	return role == RoleKguZkhAdmin || role == RoleKguZkhUser
}

func IsTooAdmin(role string) bool {
	return role == RoleLandfillAdmin
}

// IsContractorAdmin проверяет, является ли роль администратором подрядчика.
func IsContractorAdmin(role string) bool {
	return role == RoleContractorAdmin
}

// IsContractorRole проверяет, является ли роль пользователем подрядчика (admin или user).
func IsContractorRole(role string) bool {
	return role == RoleContractorAdmin || role == RoleContractorUser
}

func IsLandfillAdmin(role string) bool {
	return role == RoleLandfillAdmin
}

// IsLandfillRole проверяет, является ли роль пользователем полигона (admin или user).
func IsLandfillRole(role string) bool {
	return role == RoleLandfillAdmin || role == RoleLandfillUser
}

// IsDriver проверяет, является ли роль водителем.
func IsDriver(role string) bool {
	return role == RoleDriver
}
