package dto

type AdminMailRecipients struct {
	Type    string   `json:"type" validate:"required,oneof=all filtered selected"` // all, filtered, selected
	Role    string   `json:"role,omitempty"`                                        // filter by role
	Status  string   `json:"status,omitempty"`                                      // filter by status (active/inactive/banned)
	UserIDs []string `json:"user_ids,omitempty"`                                    // for type=selected
}

type AdminMailRequest struct {
	Subject    string              `json:"subject" validate:"required,min=1,max=255"`
	Body       string              `json:"body" validate:"required,min=1"`
	Recipients AdminMailRecipients `json:"recipients" validate:"required"`
}

type AdminMailResponse struct {
	Sent       int      `json:"sent"`
	Failed     int      `json:"failed"`
	Errors     []string `json:"errors,omitempty"`
	Recipients int      `json:"recipients"`
}
