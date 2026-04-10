package dto

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
	ID         string   `json:"id"`
	RefID      string   `json:"ref_id"`
	Type       string   `json:"type"`
	OrderIndex int      `json:"order_index"`
	AttemptID  *string  `json:"attempt_id,omitempty"`
	Score      *float64 `json:"score,omitempty"`
}

type ModuleItem struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	ElementCount int             `json:"element_count"`
	Items        []CourseItemDTO `json:"items"`
}

type CourseDetailResponse struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	TeacherName  string       `json:"teacher_name"`
	ModuleCount  int          `json:"module_count"`
	ElementCount int          `json:"element_count"`
	IsTeacher    bool         `json:"is_teacher"`
	Modules      []ModuleItem `json:"modules"`
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

// ─── Course item ──────────────────────────────────────────────────────────────

type CreateCourseItemRequest struct {
	RefID      string `json:"ref_id"`
	Type       string `json:"type"`
	OrderIndex int    `json:"order_index"`
}

type CreateCourseItemResponse struct {
	ID string `json:"id"`
}
