package cmd

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/phuangpheth/assessment/expense"
)

type handler struct {
	expenseSvc *expense.Service
}

func NewHandler(router *echo.Echo, svc *expense.Service) error {
	if router == nil || svc == nil {
		return errors.New("invalid argument")
	}
	h := handler{
		expenseSvc: svc,
	}

	router.GET("/expenses", h.ListExpenses, Auth)
	router.GET("/expenses/:id", h.GetExpenseByID, Auth)
	router.POST("/expenses", h.SaveExpense, Auth)
	router.PUT("/expenses/:id", h.UpdateExpense, Auth)
	return nil
}

func (h *handler) SaveExpense(c echo.Context) error {
	var exp expense.Expense
	if err := c.Bind(&exp); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"code":    http.StatusBadRequest,
			"message": "invalid request body",
		})
	}
	if err := exp.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
	}

	ctx := c.Request().Context()
	expense, err := h.expenseSvc.Save(ctx, &exp)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error: ",
		})
	}
	return c.JSON(http.StatusCreated, expense)
}

func (h *handler) UpdateExpense(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"code":    http.StatusBadRequest,
			"message": "invalid params",
		})
	}

	var exp expense.Expense
	if err := c.Bind(&exp); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"code":    http.StatusBadRequest,
			"message": "invalid request body",
		})
	}
	if err := exp.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
	}

	ctx := c.Request().Context()
	exp.ID = id
	ex, err := h.expenseSvc.Update(ctx, &exp)
	if errors.Is(err, expense.ErrNotFound) {
		return c.JSON(http.StatusNotFound, echo.Map{
			"code":    http.StatusNotFound,
			"message": errors.Unwrap(err).Error(),
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error: ",
		})
	}
	return c.JSON(http.StatusOK, ex)
}

func (h *handler) ListExpenses(c echo.Context) error {
	ctx := c.Request().Context()
	exps, err := h.expenseSvc.List(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error",
		})
	}
	return c.JSON(http.StatusOK, exps)
}

func (h *handler) GetExpenseByID(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"code":    http.StatusBadRequest,
			"message": "invalid params",
		})
	}
	exp, err := h.expenseSvc.GetByID(ctx, id)
	if errors.Is(err, expense.ErrNotFound) {
		return c.JSON(http.StatusNotFound, echo.Map{
			"code":    http.StatusNotFound,
			"message": errors.Unwrap(err).Error(),
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"code":    http.StatusInternalServerError,
			"message": "Internal Server Error : ",
		})
	}
	return c.JSON(http.StatusOK, exp)
}
