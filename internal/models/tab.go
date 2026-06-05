package models

import "encoding/json"

type TabManifest struct {
	ID                  string          `json:"id"`
	DisplayName         string          `json:"displayName"`
	Description         string          `json:"description,omitempty"`
	Icon                string          `json:"icon"`
	Route               string          `json:"route"`
	EntryType           string          `json:"entryType"`
	EntryURI            string          `json:"entryUri,omitempty"`
	Version             SemanticVersion `json:"version"`
	MinContainerVersion int             `json:"minContainerVersion"`
	Permissions         []string        `json:"permissions"`
	Enabled             bool            `json:"enabled"`
	SortOrder           int             `json:"sortOrder,omitempty"`
	Extension           *TabExtension   `json:"extension,omitempty"`
	ExtraConfig         json.RawMessage `json:"extraConfig,omitempty"`
	Visibility          *TabVisibility  `json:"visibility,omitempty"`
}

type SemanticVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch,omitempty"`
}

type TabExtension struct {
	TitleBar    *TitleBarExtension    `json:"titleBar,omitempty"`
	Fab         *FabExtension         `json:"fab,omitempty"`
	BottomPanel *BottomPanelExtension `json:"bottomPanel,omitempty"`
}

type TitleBarExtension struct {
	RightIcon string     `json:"rightIcon,omitempty"`
	RightText string     `json:"rightText,omitempty"`
	MenuItems []MenuItem `json:"menuItems"`
}

type MenuItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type FabExtension struct {
	ID    string `json:"id,omitempty"`
	Icon  string `json:"icon"`
	Label string `json:"label"`
}

type BottomPanelExtension struct {
	DefaultHeight int `json:"defaultHeight"`
}

type EnableTabRequest struct {
	TabID string `json:"tabId"`
}

type TabMutationResponse struct {
	Success bool   `json:"success"`
	TabID   string `json:"tabId"`
}

type CreateCustomTabRequest struct {
	ID                  string                `json:"id"`
	DisplayName         string                `json:"displayName"`
	Description         string                `json:"description,omitempty"`
	Icon                string                `json:"icon"`
	Route               string                `json:"route"`
	EntryType           string                `json:"entryType"`
	EntryURI            string                `json:"entryUri"`
	MinContainerVersion int                   `json:"minContainerVersion"`
	SortOrder           int                   `json:"sortOrder,omitempty"`
	Visibility          *TabVisibilityRequest `json:"visibility,omitempty"`
}

type CustomTabResponse struct {
	Success bool        `json:"success"`
	TabID   string      `json:"tabId,omitempty"`
	Tab     TabManifest `json:"tab,omitempty"`
}

type UpdateCustomTabRequest struct {
	DisplayName string                `json:"displayName"`
	Description string                `json:"description,omitempty"`
	Icon        string                `json:"icon"`
	EntryURI    string                `json:"entryUri"`
	SortOrder   int                   `json:"sortOrder,omitempty"`
	Visibility  *TabVisibilityRequest `json:"visibility,omitempty"`
}

type TabVisibility struct {
	Scope          string   `json:"scope"`
	TeamIDs        []string `json:"teamIds"`
	UserIDs        []string `json:"userIds"`
	DefaultEnabled bool     `json:"defaultEnabled"`
}

type TabVisibilityRequest struct {
	Scope          string   `json:"scope"`
	TeamIDs        []string `json:"teamIds"`
	UserIDs        []string `json:"userIds"`
	DefaultEnabled *bool    `json:"defaultEnabled,omitempty"`
}

type ReorderTabsRequest struct {
	Items []ReorderTabItem `json:"items"`
}

type ReorderTabItem struct {
	TabID     string `json:"tabId"`
	SortOrder int    `json:"sortOrder"`
}

type ValidateTabRequest struct {
	ContainerVersion int         `json:"containerVersion"`
	Permissions      []string    `json:"permissions"`
	Tab              TabManifest `json:"tab"`
}

type ValidationIssue struct {
	Code         string `json:"code"`
	ProtocolCode int    `json:"protocolCode,omitempty"`
	Message      string `json:"message"`
	Field        string `json:"field,omitempty"`
}

type ValidateTabResponse struct {
	Valid         bool              `json:"valid"`
	Openable      bool              `json:"openable"`
	Errors        []ValidationIssue `json:"errors"`
	Warnings      []ValidationIssue `json:"warnings"`
	NormalizedTab *TabManifest      `json:"normalizedTab"`
}

type ActionRequest struct {
	Source  string          `json:"source"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type ActionNext struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Target string `json:"target,omitempty"`
}

type ActionResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Next    ActionNext `json:"next"`
}
