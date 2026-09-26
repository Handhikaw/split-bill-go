package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type ExpenseHandler struct {
	expenseService service.ExpenseService
}

func NewExpenseHandler(expenseService service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{expenseService: expenseService}
}

func participantIDFromContext(c *gin.Context) int64 {
	if val, exists := c.Get("participant_id"); exists {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case int32:
			return int64(v)
		case uint:
			return int64(v)
		case uint64:
			return int64(v)
		}
	}

	return 0
}

func (h *ExpenseHandler) Expense(c *gin.Context) {
	var req dto.CreateExpenseRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	res, err := h.expenseService.CreateExpense(req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, model.Success(res))
}

func (h *ExpenseHandler) UpdateExpense(c *gin.Context) {
	var req dto.UpdateExpenseRq
	publicID := c.Param("public_id")

	expenseIDStr := c.Param("expense_id")
	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid expense_id"))
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	participantID := participantIDFromContext(c)

	res, err := h.expenseService.UpdateExpense(publicID, expenseID, participantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}

func (h *ExpenseHandler) ExpenseItem(c *gin.Context) {
	var req dto.CreateExpenseItemRq
	publicID := c.Param("public_id")

	expenseIDStr := c.Param("expense_id")

	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid expense_id"))
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	participantID := participantIDFromContext(c)

	res, err := h.expenseService.CreateExpenseItem(publicID, expenseID, participantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, model.Success(res))
}

func (h *ExpenseHandler) UpdateExpenseItem(c *gin.Context) {
	var req dto.UpdateExpenseItemRq
	publicID := c.Param("public_id")

	expenseIDStr := c.Param("expense_id")
	itemIDStr := c.Param("item_id")

	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid expense_id"))
		return
	}

	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid item_id"))
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	participantID := participantIDFromContext(c)

	err = h.expenseService.UpdateExpenseItem(publicID, expenseID, itemID, participantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

func (h *ExpenseHandler) DeleteExpenseItem(c *gin.Context) {
	publicID := c.Param("public_id")

	expenseIDStr := c.Param("expense_id")
	itemIDStr := c.Param("item_id")

	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid expense_id"))
		return
	}

	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid item_id"))
		return
	}

	participantID := participantIDFromContext(c)

	err = h.expenseService.DeleteExpenseItem(publicID, expenseID, itemID, participantID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

func (h *ExpenseHandler) DeleteExpense(c *gin.Context) {
	publicID := c.Param("public_id")
	expenseIDStr := c.Param("expense_id")
	participantID := participantIDFromContext(c)

	expenseID, err := strconv.ParseInt(expenseIDStr, 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid expense_id"))
		return
	}

	err = h.expenseService.DeleteExpense(publicID, expenseID, participantID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

func (h *ExpenseHandler) GetCategory(c *gin.Context) {
	userIDParam := c.Query("user_id")

	var userID *int64
	if userIDParam != "" {
		parsedUserID, err := strconv.ParseInt(userIDParam, 10, 64)
		if err != nil {
			_ = c.Error(model.NewBadRequest("invalid user_id"))
			return
		}
		userID = &parsedUserID
	}

	res, err := h.expenseService.GetCategory(userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}

func (h *ExpenseHandler) UpsertCategory(c *gin.Context) {
	var req dto.CategoryRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	if err := h.expenseService.UpsertCategory(req); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}
