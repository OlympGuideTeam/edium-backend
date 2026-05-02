package dto

import "encoding/json"

// ─── Course list ──────────────────────────────────────────────────────────────

type CourseSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	TeacherName  string `json:"teacher_name"`
	ModuleCount  int    `json:"module_count"`
	ElementCount int    `json:"element_count"`
	IsTeacher    bool   `json:"is_teacher"`
}

type CourseListResponse struct {
	Courses []CourseSummary `json:"courses"`
}

// ─── Create ───────────────────────────────────────────────────────────────────

type CreateCourseRequest struct {
	ClassID string `json:"class_id"`
	Title   string `json:"title"`
}

type CreateCourseResponse struct {
	ID string `json:"id"`
}

// ─── Update ───────────────────────────────────────────────────────────────────

type UpdateCourseRequest struct {
	Title string `json:"title"`
}

// ─── Detail ───────────────────────────────────────────────────────────────────

type CourseItemDTO struct {
	ID        string          `json:"id"`
	ObjectID  string          `json:"object_id"`
	Type      string          `json:"type"`
	AttemptID *string         `json:"attempt_id,omitempty"`
	Score     *float64        `json:"score,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type ModuleItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ElementCount int    `json:"element_count"`
}

type ModuleDetailResponse struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	ElementCount int             `json:"element_count"`
	Items        []CourseItemDTO `json:"items"`
}

type ModuleRosterResponse struct {
	Members []MemberShort `json:"members"`
}

type CourseDraftDTO struct {
	ID             string          `json:"id"`
	QuizTemplateID string          `json:"quiz_template_id"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
}

type CourseDetailResponse struct {
	ID           string           `json:"id"`
	Title        string           `json:"title"`
	TeacherName  string           `json:"teacher_name"`
	ModuleCount  int              `json:"module_count"`
	ElementCount int              `json:"element_count"`
	IsTeacher    bool             `json:"is_teacher"`
	Drafts       []CourseDraftDTO `json:"drafts"`
	Modules      []ModuleItem     `json:"modules"`
}

// ─── Module ───────────────────────────────────────────────────────────────────

type CreateModuleRequest struct {
	Title string `json:"title"`
}

type CreateModuleResponse struct {
	ID string `json:"id"`
}

type UpdateModuleRequest struct {
	Title string `json:"title"`
}

type ReorderModulesRequest struct {
	ModuleIDs []string `json:"module_ids" binding:"required,min=1"`
}

// ─── Course item ──────────────────────────────────────────────────────────────

type CreateCourseItemRequest struct {
	ObjectID string `json:"object_id"`
	Type     string `json:"type"`
}

type CreateCourseItemResponse struct {
	ID string `json:"id"`
}
