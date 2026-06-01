package repositories

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"opentab-server/internal/database"
	"opentab-server/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func mapGormError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func mapCreateError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "duplicate key") || strings.Contains(message, "unique constraint") {
		return ErrConflict
	}
	return err
}

func tabRecordToManifest(record database.TabRecord, enabled bool, sortOrder int) (models.TabManifest, error) {
	var permissions []string
	if len(record.PermissionsJSON) > 0 {
		if err := json.Unmarshal(record.PermissionsJSON, &permissions); err != nil {
			return models.TabManifest{}, err
		}
	}

	var extension *models.TabExtension
	if len(record.ExtensionJSON) > 0 {
		var parsed models.TabExtension
		if err := json.Unmarshal(record.ExtensionJSON, &parsed); err != nil {
			return models.TabManifest{}, err
		}
		extension = &parsed
	}

	return models.TabManifest{
		ID:                  record.ID,
		DisplayName:         record.DisplayName,
		Description:         record.Description,
		Icon:                record.Icon,
		Route:               record.Route,
		EntryType:           record.EntryType,
		EntryURI:            record.EntryURI,
		Version:             models.SemanticVersion{Major: record.VersionMajor, Minor: record.VersionMinor, Patch: record.VersionPatch},
		MinContainerVersion: record.MinContainerVersion,
		Permissions:         permissions,
		Enabled:             enabled,
		SortOrder:           sortOrder,
		Extension:           extension,
		ExtraConfig:         []byte(record.ExtraConfigJSON),
	}, nil
}

func manifestToTabRecord(tab models.TabManifest, userID string, isSystem bool) (database.TabRecord, error) {
	permissionsJSON, err := json.Marshal(tab.Permissions)
	if err != nil {
		return database.TabRecord{}, err
	}

	var extensionJSON datatypes.JSON
	if tab.Extension != nil {
		data, err := json.Marshal(tab.Extension)
		if err != nil {
			return database.TabRecord{}, err
		}
		extensionJSON = datatypes.JSON(data)
	}

	var owner *string
	if userID != "" {
		owner = &userID
	}

	return database.TabRecord{
		ID:                  tab.ID,
		OwnerUserID:         owner,
		DisplayName:         tab.DisplayName,
		Description:         tab.Description,
		Icon:                tab.Icon,
		Route:               tab.Route,
		EntryType:           tab.EntryType,
		EntryURI:            tab.EntryURI,
		VersionMajor:        tab.Version.Major,
		VersionMinor:        tab.Version.Minor,
		VersionPatch:        tab.Version.Patch,
		MinContainerVersion: tab.MinContainerVersion,
		PermissionsJSON:     datatypes.JSON(permissionsJSON),
		ExtensionJSON:       extensionJSON,
		ExtraConfigJSON:     datatypes.JSON(tab.ExtraConfig),
		IsSystem:            isSystem,
	}, nil
}

func formatTime(value time.Time) string {
	return value.Format(time.RFC3339)
}
