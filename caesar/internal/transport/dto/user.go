package dto

type UserProfileResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

type UpdateUserRequest struct {
	Name    *string `json:"name"`
	Surname *string `json:"surname"`
}

type UsersRosterRequest struct {
	UserIDs []string `json:"user_ids"`
}

type UsersRosterResponse struct {
	Users []UserProfileResponse `json:"users"`
}

type UserStatisticResponse struct {
	ClassTeacherCount  int `json:"class_teacher_count"`
	StudentCount       int `json:"student_count"`
	CourseTeacherCount int `json:"course_teacher_count"`
	CourseStudentCount int `json:"course_student_count"`
}
