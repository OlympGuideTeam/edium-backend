package attempt

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

func (h *Handler) GetMeStatistic(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	st, err := h.service.GetUserStatistic(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.UserStatisticResponse{
		QuizCountPassed:       st.QuizCountPassed,
		AvgQuizScore:          st.AvgQuizScore,
		QuizSessionsConducted: st.QuizSessionsConducted,
	})
}
