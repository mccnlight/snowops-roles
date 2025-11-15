package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/MSTimX/Snowops-roles/internal/database"
	"github.com/MSTimX/Snowops-roles/internal/models"
)

type CreateOrganizationRequest struct {
	Name          string `json:"name" binding:"required"`
	Type          string `json:"type" binding:"required"`
	BIN           string `json:"bin"`
	HeadFullName  string `json:"head_full_name"`
	Address       string `json:"address"`
	Phone         string `json:"phone"`
	AdminFullName string `json:"admin_full_name"`
	AdminPhone    string `json:"admin_phone"`
	AdminPassword string `json:"admin_password"`
}

type CreateDriverRequest struct {
	FullName  string `json:"full_name" binding:"required"`
	IIN       string `json:"iin" binding:"required"`
	BirthYear int    `json:"birth_year" binding:"required"`
	Phone     string `json:"phone" binding:"required"`
}

type CreateVehicleRequest struct {
	PlateNumber  string  `json:"plate_number" binding:"required"`
	Brand        string  `json:"brand" binding:"required"`
	Model        string  `json:"model" binding:"required"`
	Color        string  `json:"color" binding:"required"`
	Year         int     `json:"year" binding:"required"`
	BodyVolumeM3 float64 `json:"body_volume_m3" binding:"required"`
	PhotoURL     *string `json:"photo_url"`
	DriverID     *string `json:"driver_id"`
	IsActive     *bool   `json:"is_active"`
}

type UpdateVehicleRequest struct {
	PlateNumber  *string  `json:"plate_number"`
	Brand        *string  `json:"brand"`
	Model        *string  `json:"model"`
	Color        *string  `json:"color"`
	Year         *int     `json:"year"`
	BodyVolumeM3 *float64 `json:"body_volume_m3"`
	PhotoURL     *string  `json:"photo_url"`
	DriverID     *string  `json:"driver_id"`
	IsActive     *bool    `json:"is_active"`
}

// RegisterRoutes регистрирует HTTP-маршруты для API.
func RegisterRoutes(api *gin.RouterGroup) {
	protected := api.Group("")
	protected.Use(roleManagementGuard())

	organizations := protected.Group("/organizations")
	organizations.GET("", ListOrganizations)
	organizations.POST("", CreateOrganization)
	organizations.GET("/:id", GetOrganization)
	organizations.PUT("/:id", UpdateOrganization)
	organizations.DELETE("/:id", DeleteOrganization)

	users := protected.Group("/users")
	users.GET("", FindUser)
	users.GET("/:id", GetUser)
	users.PUT("/:id", UpdateUser)

	drivers := protected.Group("/drivers")
	drivers.GET("", ListDrivers)
	drivers.POST("", CreateDriver)
	drivers.GET("/:id", GetDriver)
	drivers.PUT("/:id", UpdateDriver)
	drivers.DELETE("/:id", DeleteDriver)

	vehicles := protected.Group("/vehicles")
	vehicles.GET("", ListVehicles)
	vehicles.POST("", CreateVehicle)
	vehicles.GET("/:id", GetVehicle)
	vehicles.PATCH("/:id", UpdateVehicle)
	vehicles.DELETE("/:id", DeleteVehicle)
}

func ListOrganizations(c *gin.Context) {
	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	currentOrgUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var orgs []models.Organization

	switch role {
	case models.RoleAkimatAdmin:
		if err := database.DB.Where("is_active = ?", true).Find(&orgs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch organizations"})
			return
		}
	case models.RoleKguZkhAdmin:
		var currentOrg models.Organization
		if err := database.DB.Where("id = ? AND is_active = ?", currentOrgUUID, true).First(&currentOrg).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch organization"})
			}
			return
		}

		orgs = append(orgs, currentOrg)

		var contractors []models.Organization
		if err := database.DB.Where("parent_org_id = ? AND type = ? AND is_active = ?", currentOrgUUID, models.OrgTypeContractor, true).Find(&contractors).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch contractor organizations"})
			return
		}

		orgs = append(orgs, contractors...)
	case models.RoleTooAdmin, models.RoleContractorAdmin:
		var currentOrg models.Organization
		if err := database.DB.Where("id = ? AND is_active = ?", currentOrgUUID, true).First(&currentOrg).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch organization"})
			}
			return
		}
		orgs = append(orgs, currentOrg)
	case models.RoleDriver:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

func CreateOrganization(c *gin.Context) {
	currentUserID := c.GetString("currentUserID")
	currentUserRole := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if currentUserID == "" || currentUserRole == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.AdminPhone = strings.TrimSpace(req.AdminPhone)
	if !models.CanCreateOrganization(currentUserRole, req.Type) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	currentOrgUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	if req.AdminPhone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin_phone is required"})
		return
	}

	req.BIN = strings.TrimSpace(req.BIN)

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if req.BIN != "" {
		inUse, err := organizationBINExists(tx, req.BIN, nil)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate BIN"})
			return
		}
		if inUse {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "organization with this BIN already exists"})
			return
		}
	}

	inUse, err := userPhoneExists(tx, req.AdminPhone, nil)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate phone"})
		return
	}
	if inUse {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "admin phone already in use"})
		return
	}

	parentOrgID := currentOrgUUID
	org := models.Organization{
		Type:         req.Type,
		Name:         req.Name,
		BIN:          req.BIN,
		HeadFullName: req.HeadFullName,
		Address:      req.Address,
		Phone:        req.Phone,
		ParentOrgID:  &parentOrgID,
		IsActive:     true,
	}

	if err := tx.Create(&org).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
		return
	}

	var adminRole string
	switch req.Type {
	case models.OrgTypeKguZkh:
		adminRole = models.RoleKguZkhAdmin
	case models.OrgTypeToo:
		adminRole = models.RoleTooAdmin
	case models.OrgTypeContractor:
		adminRole = models.RoleContractorAdmin
	default:
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported organization type"})
		return
	}

	user := models.User{
		Phone:          req.AdminPhone,
		Role:           adminRole,
		OrganizationID: &org.ID,
		IsActive:       true,
	}

	if req.AdminPassword != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		password := string(hashed)
		user.PasswordHash = &password
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin user"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"organization": gin.H{
			"id":           org.ID,
			"name":         org.Name,
			"type":         org.Type,
			"bin":          org.BIN,
			"headFullName": org.HeadFullName,
			"address":      org.Address,
			"phone":        org.Phone,
			"parentOrgID":  org.ParentOrgID,
			"isActive":     org.IsActive,
			"createdAt":    org.CreatedAt,
			"updatedAt":    org.UpdatedAt,
		},
		"admin": gin.H{
			"id":             user.ID,
			"phone":          user.Phone,
			"role":           user.Role,
			"organizationID": user.OrganizationID,
			"isActive":       user.IsActive,
			"createdAt":      user.CreatedAt,
			"updatedAt":      user.UpdatedAt,
		},
	})
}

func GetOrganization(c *gin.Context) {
	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	currentOrgUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	targetID := c.Param("id")
	orgUUID, err := uuid.Parse(targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var org models.Organization
	if err := database.DB.Where("id = ? AND is_active = ?", orgUUID, true).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch organization"})
		}
		return
	}

	switch role {
	case models.RoleAkimatAdmin:
	case models.RoleKguZkhAdmin:
		if org.ID != currentOrgUUID {
			if org.Type != models.OrgTypeContractor || org.ParentOrgID == nil || *org.ParentOrgID != currentOrgUUID {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		}
	case models.RoleTooAdmin, models.RoleContractorAdmin:
		if org.ID != currentOrgUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	case models.RoleDriver:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

func UpdateOrganization(c *gin.Context) {
	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	currentOrgUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	targetID := c.Param("id")
	orgUUID, err := uuid.Parse(targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var org models.Organization
	if err := database.DB.Where("id = ? AND is_active = ?", orgUUID, true).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch organization"})
		}
		return
	}

	// Проверка прав доступа
	switch role {
	case models.RoleAkimatAdmin:
	case models.RoleKguZkhAdmin:
		if org.ID != currentOrgUUID {
			if org.Type != models.OrgTypeContractor || org.ParentOrgID == nil || *org.ParentOrgID != currentOrgUUID {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		}
	case models.RoleTooAdmin, models.RoleContractorAdmin:
		if org.ID != currentOrgUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	case models.RoleDriver:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var body struct {
		Name         *string `json:"name"`
		Type         *string `json:"type"`
		BIN          *string `json:"bin"`
		HeadFullName *string `json:"head_full_name"`
		Address      *string `json:"address"`
		Phone        *string `json:"phone"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if body.Type != nil && strings.TrimSpace(*body.Type) != "" && strings.TrimSpace(*body.Type) != org.Type {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization type cannot be changed"})
		return
	}

	updates := map[string]interface{}{}

	if body.Name != nil {
		updates["name"] = strings.TrimSpace(*body.Name)
	}
	if body.HeadFullName != nil {
		updates["head_full_name"] = strings.TrimSpace(*body.HeadFullName)
	}
	if body.Address != nil {
		updates["address"] = strings.TrimSpace(*body.Address)
	}
	if body.Phone != nil {
		updates["phone"] = strings.TrimSpace(*body.Phone)
	}
	if body.BIN != nil {
		binValue := strings.TrimSpace(*body.BIN)
		if binValue != "" {
			inUse, err := organizationBINExists(database.DB, binValue, &org.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate BIN"})
				return
			}
			if inUse {
				c.JSON(http.StatusConflict, gin.H{"error": "organization with this BIN already exists"})
				return
			}
			updates["bin"] = binValue
		} else {
			updates["bin"] = nil
		}
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&org).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update organization"})
		return
		}
	}

	if err := database.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updated organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

func DeleteOrganization(c *gin.Context) {
	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	currentOrgUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	targetID := c.Param("id")
	orgUUID, err := uuid.Parse(targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var org models.Organization
	if err := database.DB.Where("id = ? AND is_active = ?", orgUUID, true).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch organization"})
		}
		return
	}

	switch role {
	case models.RoleAkimatAdmin:
	case models.RoleKguZkhAdmin:
		if org.ID != currentOrgUUID {
			if org.Type != models.OrgTypeContractor || org.ParentOrgID == nil || *org.ParentOrgID != currentOrgUUID {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
		}
	case models.RoleTooAdmin, models.RoleContractorAdmin:
		if org.ID != currentOrgUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}

	if err := tx.Model(&org).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate organization"})
		return
	}

	if err := tx.Model(&models.User{}).Where("organization_id = ?", org.ID).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate organization users"})
		return
	}

	if org.Type == models.OrgTypeContractor {
		if err := tx.Model(&models.Driver{}).Where("contractor_id = ?", org.ID).Update("is_active", false).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate drivers"})
			return
		}

		var driverIDs []uuid.UUID
		if err := tx.Model(&models.Driver{}).Where("contractor_id = ?", org.ID).Pluck("id", &driverIDs).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch driver ids"})
			return
		}

		if len(driverIDs) > 0 {
			if err := tx.Model(&models.User{}).Where("driver_id IN ?", driverIDs).Update("is_active", false).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate driver users"})
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finalize organization deletion"})
		return
	}

	c.Status(http.StatusNoContent)
}

func FindUser(c *gin.Context) {
	phone := c.Query("phone")
	login := c.Query("login")

	if phone == "" && login == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone or login required"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var user models.User
	q := database.DB.Model(&models.User{}).Where("is_active = ?", true)

	if phone != "" {
		q = q.Where("phone = ?", phone)
	}
	if login != "" {
		q = q.Where("login = ?", login)
	}

	if err := q.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func GetUser(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	role := c.GetString("currentUserRole")
	currentUserID := c.GetString("currentUserID")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentUserID == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	targetID := c.Param("id")
	targetUUID, err := uuid.Parse(targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ? AND is_active = ?", targetUUID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	currentUserUUID, err := uuid.Parse(currentUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current user id"})
		return
	}

	if user.ID == currentUserUUID {
		c.JSON(http.StatusOK, gin.H{"user": user})
		return
	}

	currentOrgUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	if !canAccessUser(role, currentOrgUUID, &user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func UpdateUser(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	role := c.GetString("currentUserRole")
	currentUserID := c.GetString("currentUserID")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentUserID == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	targetID := c.Param("id")
	targetUUID, err := uuid.Parse(targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ? AND is_active = ?", targetUUID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	currentUserUUID, err := uuid.Parse(currentUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current user id"})
		return
	}

	currentOrgUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	// Проверка прав доступа
	if user.ID != currentUserUUID {
		if !canAccessUser(role, currentOrgUUID, &user) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	var body struct {
		Phone          *string    `json:"phone"`
		Login          *string    `json:"login"`
		Password       *string    `json:"password"`
		Role           *string    `json:"role"`
		OrganizationID *uuid.UUID `json:"organization_id"`
		DriverID       *uuid.UUID `json:"driver_id"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	updateData := make(map[string]interface{})
	if body.Phone != nil {
		updateData["phone"] = *body.Phone
	}
	if body.Login != nil {
		updateData["login"] = *body.Login
	}
	if body.Password != nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*body.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		passwordHash := string(hashed)
		updateData["password_hash"] = passwordHash
	}
	if body.Role != nil {
		updateData["role"] = *body.Role
	}
	if body.OrganizationID != nil {
		updateData["organization_id"] = *body.OrganizationID
	}
	if body.DriverID != nil {
		updateData["driver_id"] = *body.DriverID
	}

	if len(updateData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	if err := database.DB.Model(&user).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
		return
	}

	if err := database.DB.Where("id = ?", targetUUID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func ListDrivers(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	currentOrgUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	var drivers []models.Driver

	switch role {
	case models.RoleAkimatAdmin:
		if err := database.DB.Where("is_active = ?", true).Find(&drivers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch drivers"})
			return
		}
	case models.RoleKguZkhAdmin:
		contractorIDs, err := contractorIDsForKgu(currentOrgUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve contractor organizations"})
			return
		}
		contractorIDs = append(contractorIDs, currentOrgUUID)
		if len(contractorIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"drivers": []models.Driver{}})
			return
		}
		if err := database.DB.Where("is_active = ? AND contractor_id IN ?", true, contractorIDs).Find(&drivers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch drivers"})
			return
		}
	case models.RoleContractorAdmin:
		if err := database.DB.Where("is_active = ? AND contractor_id = ?", true, currentOrgUUID).Find(&drivers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch drivers"})
			return
		}
	case models.RoleDriver:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"drivers": drivers})
}

func CreateDriver(c *gin.Context) {
	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if role != models.RoleContractorAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req CreateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone is required"})
		return
	}

	contractorUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	inUse, err := userPhoneExists(tx, req.Phone, nil)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate phone"})
		return
	}
	if inUse {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "phone already in use"})
		return
	}

	contractorID := contractorUUID
	driver := models.Driver{
		ContractorID: &contractorID,
		FullName:     req.FullName,
		IIN:          req.IIN,
		BirthYear:    req.BirthYear,
		Phone:        req.Phone,
		IsActive:     true,
	}

	if err := tx.Create(&driver).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create driver"})
		return
	}

	driverID := driver.ID
	user := models.User{
		Phone:          req.Phone,
		Role:           models.RoleDriver,
		OrganizationID: &contractorID,
		DriverID:       &driverID,
		IsActive:       true,
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create driver user"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"driver": driver,
		"user": gin.H{
			"id":             user.ID,
			"phone":          user.Phone,
			"role":           user.Role,
			"organizationID": user.OrganizationID,
			"driverID":       user.DriverID,
			"isActive":       user.IsActive,
			"createdAt":      user.CreatedAt,
			"updatedAt":      user.UpdatedAt,
		},
	})
}

func GetDriver(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id := c.Param("id")

	var driver models.Driver
	if err := database.DB.Where("id = ? AND is_active = ?", id, true).First(&driver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "driver not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	role := c.GetString("currentUserRole")
	orgID := c.GetString("currentOrgID")

	contractorOrgID := ""
	if driver.ContractorID != nil {
		contractorOrgID = driver.ContractorID.String()
	}

	if !CanAccessDriver(role, orgID, contractorOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"driver": driver})
}

func UpdateDriver(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id := c.Param("id")

	var driver models.Driver
	if err := database.DB.Where("id = ?", id).First(&driver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "driver not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	var driverUser models.User
	driverUserExists := false
	if err := database.DB.Where("driver_id = ?", driver.ID).First(&driverUser).Error; err == nil {
		driverUserExists = true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch driver user"})
		return
	}

	role := c.GetString("currentUserRole")
	orgID := c.GetString("currentOrgID")

	contractorOrgID := ""
	if driver.ContractorID != nil {
		contractorOrgID = driver.ContractorID.String()
	}

	if !CanAccessDriver(role, orgID, contractorOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	var body struct {
		FullName  *string `json:"full_name"`
		Phone     *string `json:"phone"`
		BirthYear *int    `json:"birth_year"`
		IIN       *string `json:"iin"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	updates := map[string]interface{}{}

	if body.FullName != nil {
		updates["full_name"] = strings.TrimSpace(*body.FullName)
	}
	if body.BirthYear != nil {
		updates["birth_year"] = *body.BirthYear
	}
	if body.IIN != nil {
		updates["iin"] = strings.TrimSpace(*body.IIN)
	}
	if body.Phone != nil {
		newPhone := strings.TrimSpace(*body.Phone)
		if newPhone == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phone cannot be empty"})
			return
		}
		var excludeID *uuid.UUID
		if driverUserExists {
			excludeID = &driverUser.ID
		}
		inUse, err := userPhoneExists(database.DB, newPhone, excludeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate phone"})
			return
		}
		if inUse {
			c.JSON(http.StatusConflict, gin.H{"error": "phone already in use"})
			return
		}
		updates["phone"] = newPhone
		if err := database.DB.Model(&models.User{}).Where("driver_id = ?", driver.ID).Update("phone", newPhone).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update driver user"})
			return
		}
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&driver).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
		return
		}
	}

	if err := database.DB.Where("id = ?", id).First(&driver).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"driver": driver})
}

func DeleteDriver(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id := c.Param("id")

	var driver models.Driver
	if err := database.DB.Where("id = ?", id).First(&driver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "driver not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	role := c.GetString("currentUserRole")
	orgID := c.GetString("currentOrgID")

	contractorOrgID := ""
	if driver.ContractorID != nil {
		contractorOrgID = driver.ContractorID.String()
	}

	if !CanAccessDriver(role, orgID, contractorOrgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := database.DB.Model(&driver).Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
		return
	}

	if err := database.DB.Model(&models.User{}).Where("driver_id = ?", driver.ID).Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
		return
	}

	if err := database.DB.Model(&models.Vehicle{}).Where("driver_id = ?", driver.ID).Updates(map[string]interface{}{
		"driver_id": nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
		return
	}

	c.Status(http.StatusNoContent)
}

func ListVehicles(c *gin.Context) {
	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	query := database.DB.Model(&models.Vehicle{})

	if onlyActiveEnabled(c.Query("only_active")) {
		query = query.Where("is_active = ?", true)
	}

	switch role {
	case models.RoleAkimatAdmin:
	case models.RoleKguZkhAdmin:
		currentOrgUUID, err := uuid.Parse(currentOrgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
			return
		}
		contractorIDs, err := contractorIDsForKgu(currentOrgUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve contractor organizations"})
			return
		}
		if len(contractorIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"vehicles": []models.Vehicle{}})
			return
		}
		query = query.Where("contractor_id IN ?", contractorIDs)
	case models.RoleContractorAdmin:
		contractorUUID, err := uuid.Parse(currentOrgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
			return
		}
		query = query.Where("contractor_id = ?", contractorUUID)
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var vehicles []models.Vehicle
	if err := query.Order("created_at DESC").Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vehicles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vehicles": vehicles})
}

func CreateVehicle(c *gin.Context) {
	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")

	if role == "" || currentOrgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if role != models.RoleContractorAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	contractorUUID, err := uuid.Parse(currentOrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current organization id"})
		return
	}

	var req CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Year <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "year must be positive"})
		return
	}
	if req.BodyVolumeM3 <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body_volume_m3 must be positive"})
		return
	}

	vehicle := models.Vehicle{
		ContractorID: &contractorUUID,
		PlateNumber:  strings.TrimSpace(req.PlateNumber),
		Brand:        strings.TrimSpace(req.Brand),
		Model:        strings.TrimSpace(req.Model),
		Color:        strings.TrimSpace(req.Color),
		Year:         req.Year,
		BodyVolumeM3: req.BodyVolumeM3,
		IsActive:     req.IsActive == nil || *req.IsActive,
	}
	vehicle.PhotoURL = normalizeOptionalString(req.PhotoURL)

	if err := database.DB.Create(&vehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create vehicle"})
		return
	}

	if req.DriverID != nil && strings.TrimSpace(*req.DriverID) != "" {
		if err := assignDriverToVehicle(&vehicle, strings.TrimSpace(*req.DriverID)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := database.DB.Where("id = ?", vehicle.ID).First(&vehicle).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reload vehicle"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"vehicle": vehicle})
}

func GetVehicle(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id := c.Param("id")
	var vehicle models.Vehicle
	if err := database.DB.Where("id = ?", id).First(&vehicle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")
	if !CanAccessVehicle(role, currentOrgID, vehicle.ContractorID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vehicle": vehicle})
}

func UpdateVehicle(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id := c.Param("id")
	var vehicle models.Vehicle
	if err := database.DB.Where("id = ?", id).First(&vehicle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")
	if role != models.RoleContractorAdmin || !CanAccessVehicle(role, currentOrgID, vehicle.ContractorID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req UpdateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}

	if req.PlateNumber != nil {
		updates["plate_number"] = strings.TrimSpace(*req.PlateNumber)
	}
	if req.Brand != nil {
		updates["brand"] = strings.TrimSpace(*req.Brand)
	}
	if req.Model != nil {
		updates["model"] = strings.TrimSpace(*req.Model)
	}
	if req.Color != nil {
		updates["color"] = strings.TrimSpace(*req.Color)
	}
	if req.Year != nil {
		if *req.Year <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "year must be positive"})
			return
		}
		updates["year"] = *req.Year
	}
	if req.BodyVolumeM3 != nil {
		if *req.BodyVolumeM3 <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body_volume_m3 must be positive"})
			return
		}
		updates["body_volume_m3"] = *req.BodyVolumeM3
	}
	if req.PhotoURL != nil {
		updates["photo_url"] = normalizeOptionalString(req.PhotoURL)
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		if !*req.IsActive {
			updates["driver_id"] = nil
		}
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&vehicle).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update vehicle"})
			return
		}
	}

	if req.DriverID != nil {
		if err := assignDriverToVehicle(&vehicle, strings.TrimSpace(*req.DriverID)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if err := database.DB.Where("id = ?", vehicle.ID).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reload vehicle"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vehicle": vehicle})
}

func DeleteVehicle(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id := c.Param("id")
	var vehicle models.Vehicle
	if err := database.DB.Where("id = ?", id).First(&vehicle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db query failed"})
		}
		return
	}

	role := c.GetString("currentUserRole")
	currentOrgID := c.GetString("currentOrgID")
	if role != models.RoleContractorAdmin || !CanAccessVehicle(role, currentOrgID, vehicle.ContractorID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := database.DB.Model(&vehicle).Updates(map[string]interface{}{
		"is_active": false,
		"driver_id": nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate vehicle"})
		return
	}

	c.Status(http.StatusNoContent)
}

func assignDriverToVehicle(vehicle *models.Vehicle, driverID string) error {
	if strings.TrimSpace(driverID) == "" {
		if err := database.DB.Model(vehicle).Update("driver_id", nil).Error; err != nil {
			return fmt.Errorf("failed to unassign driver")
		}
		return nil
	}

	driverUUID, err := uuid.Parse(driverID)
	if err != nil {
		return fmt.Errorf("invalid driver_id")
	}

	if vehicle.ContractorID == nil {
		return fmt.Errorf("vehicle contractor is not set")
	}

	var driver models.Driver
	if err := database.DB.Where("id = ? AND is_active = ?", driverUUID, true).First(&driver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("driver not found or inactive")
		}
		return fmt.Errorf("failed to fetch driver")
	}

	if driver.ContractorID == nil || driver.ContractorID.String() != vehicle.ContractorID.String() {
		return fmt.Errorf("driver belongs to another contractor")
	}

	if err := database.DB.Model(&models.Vehicle{}).
		Where("driver_id = ? AND contractor_id = ?", driverUUID, vehicle.ContractorID).
		Update("driver_id", nil).Error; err != nil {
		return fmt.Errorf("failed to unassign driver from previous vehicle")
	}

	if err := database.DB.Model(vehicle).Update("driver_id", driverUUID).Error; err != nil {
		return fmt.Errorf("failed to assign driver to vehicle")
	}

	return nil
}

func onlyActiveEnabled(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

func CanAccessVehicle(role, currentOrgID string, contractorID *uuid.UUID) bool {
	if role == models.RoleAkimatAdmin {
		return true
	}
	if contractorID == nil {
		return false
	}
	switch role {
	case models.RoleKguZkhAdmin:
		return true
	case models.RoleContractorAdmin:
		return currentOrgID == contractorID.String()
	default:
		return false
	}
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	result := trimmed
	return &result
}

func organizationBINExists(db *gorm.DB, bin string, excludeID *uuid.UUID) (bool, error) {
	query := db.Model(&models.Organization{}).Where("bin = ?", bin)
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func userPhoneExists(db *gorm.DB, phone string, excludeID *uuid.UUID) (bool, error) {
	query := db.Model(&models.User{}).Where("phone = ?", phone)
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func CanAccessDriver(role, currentOrgID, contractorOrgID string) bool {
	if role == "AKIMAT_ADMIN" {
		return true
	}
	if role == models.RoleKguZkhAdmin && contractorOrgID != "" {
		return true
	}
	if role == "CONTRACTOR_ADMIN" && currentOrgID == contractorOrgID {
		return true
	}
	return false
}

func canAccessUser(role string, currentOrgID uuid.UUID, user *models.User) bool {
	switch role {
	case models.RoleAkimatAdmin:
		return true
	case models.RoleKguZkhAdmin:
		// allow users belonging to current KGU or its contractors
		if user.OrganizationID != nil && *user.OrganizationID == currentOrgID {
			return true
		}
		if user.OrganizationID == nil {
			return false
		}
		orgID := *user.OrganizationID
		var org models.Organization
		if err := database.DB.Where("id = ? AND is_active = ?", orgID, true).First(&org).Error; err != nil {
			return false
		}
		if org.ParentOrgID != nil && *org.ParentOrgID == currentOrgID {
			return true
		}
	case models.RoleTooAdmin, models.RoleContractorAdmin:
		if user.OrganizationID != nil && *user.OrganizationID == currentOrgID {
			return true
		}
	}
	return false
}

func contractorIDsForKgu(kguID uuid.UUID) ([]uuid.UUID, error) {
	var orgs []models.Organization
	if err := database.DB.Where("parent_org_id = ? AND type = ? AND is_active = ?", kguID, models.OrgTypeContractor, true).Find(&orgs).Error; err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(orgs))
	for _, org := range orgs {
		ids = append(ids, org.ID)
	}
	return ids, nil
}

func roleManagementGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("currentUserRole")
		if role == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		if role == models.RoleTooAdmin || role == models.RoleDriver {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}

		c.Next()
	}
}
