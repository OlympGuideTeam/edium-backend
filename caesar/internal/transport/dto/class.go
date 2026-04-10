package dto

// ─── Classes/me ───────────────────────────────────────────────────────────────

type ClassSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	OwnerName    string `json:"owner_name"`
	StudentCount int    `json:"student_count"`
	IsOwner      bool   `json:"is_owner"`
}

type ClassListResponse struct {
	Classes []ClassSummary `json:"classes"`
}

// ─── Create ───────────────────────────────────────────────────────────────────

type CreateClassRequest struct {
	Title string `json:"title"`
}

type CreateClassResponse struct {
	ID string `json:"id"`
}

// ─── Update ───────────────────────────────────────────────────────────────────

type UpdateClassRequest struct {
	Title string `json:"title"`
}

// ─── Detail ───────────────────────────────────────────────────────────────────

type MemberShort struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

type ClassDetailResponse struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	OwnerName string          `json:"owner_name"`
	IsOwner   bool            `json:"is_owner"`
	Teachers  []MemberShort   `json:"teachers"`
	Students  []MemberShort   `json:"students"`
	Courses   []CourseSummary `json:"courses"`
}

// ─── Invitation ───────────────────────────────────────────────────────────────

type InviteResponse struct {
	InvitationID string `json:"invitation_id"`
}
