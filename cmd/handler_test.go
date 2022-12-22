package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/phuangpheth/assessment/expense"
	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	t.Run("RETURNING VALUE FROM env", func(t *testing.T) {
		t.Setenv("PORT", "8000")
		want := "8000"

		got := getEnv("PORT", "3000")

		assert.Equal(t, want, got)
	})

	t.Run("RETURNING VALUE FROM fallback", func(t *testing.T) {
		want := "3000"

		got := getEnv("PORT", "3000")

		assert.Equal(t, want, got)
	})
}

func TestNewHandler(t *testing.T) {
	t.Run("NewHandler()", func(t *testing.T) {
		e := echo.New()
		svc := &expense.Service{}
		err := NewHandler(e, svc)
		assert.NoError(t, err)
	})

	t.Run("NewHandler() returns invalid argument", func(t *testing.T) {
		want := "invalid argument"

		err := NewHandler(nil, nil)
		assert.EqualError(t, err, want)
	})
}

func TestHandlerSaveExpense(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []string{"id", "amount", "title", "note", "tags"}
	e := echo.New()
	svc := expense.NewService(db)
	h := &handler{svc}
	t.Run("SaveExpense()", func(t *testing.T) {
		exp := expense.Expense{
			ID:     1,
			Amount: 75,
			Title:  "Halo Kitty",
			Note:   "buy tea and coffee",
			Tags:   []string{"drinks", "juices"},
		}

		rows := sqlmock.NewRows(columns).AddRow(exp.ID, exp.Amount, exp.Title, exp.Note, pq.Array(exp.Tags))
		mock.ExpectQuery(`INSERT INTO expenses (.+) RETURNING`).WillReturnRows(rows)

		byt, _ := json.Marshal(exp)
		req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(string(byt)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		want := `{"id":1,"amount":75,"title":"Halo Kitty","note":"buy tea and coffee","tags":["drinks","juices"]}`

		err = h.SaveExpense(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusCreated, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})

	t.Run("SaveExpense() returns invalid request body", func(t *testing.T) {
		body := `
			{
				"amount": "79",
				"title": "strawberry smoothie",
				"note": "night market promotion discount 10 bath",
				"tags": ""
			}
		`
		req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		want := `{"code":400,"message":"invalid request body"}`

		err = h.SaveExpense(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})
}

func TestHandlerUpdateExpense(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []string{"id", "amount", "title", "note", "tags"}
	e := echo.New()
	svc := expense.NewService(db)
	h := &handler{svc}

	t.Run("UpdateExpense()", func(t *testing.T) {
		exp := expense.Expense{
			ID:     1,
			Amount: 75,
			Title:  "Halo Kitty",
			Note:   "buy tea",
			Tags:   []string{"drinks", "juices"},
		}

		rows := sqlmock.NewRows(columns).AddRow(exp.ID, exp.Amount, exp.Title, exp.Note, pq.Array(exp.Tags))
		mock.ExpectQuery("SELECT (.+) FROM expenses").WithArgs(exp.ID).WillReturnRows(rows)

		mock.ExpectExec(`UPDATE expenses`).
			WithArgs(exp.Amount, exp.Title, exp.Note, pq.Array(exp.Tags), exp.ID).
			WillReturnResult(sqlmock.NewResult(exp.ID, 1))

		byt, _ := json.Marshal(exp)
		req := httptest.NewRequest(http.MethodPost, "/expenses/:id", strings.NewReader(string(byt)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		want := `{"id":1,"amount":75,"title":"Halo Kitty","note":"buy tea","tags":["drinks","juices"]}`

		err = h.UpdateExpense(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})

	t.Run("UpdateExpense() returns invalid params", func(t *testing.T) {
		body := `
			{
				"amount": 79,
				"title": "strawberry smoothie",
				"note": "night market promotion discount 10 bath",
				"tags": []
			}
		`
		req := httptest.NewRequest(http.MethodPost, "/expenses/:id", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("A")
		want := `{"code":400,"message":"invalid params"}`

		err = h.UpdateExpense(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})

	t.Run("UpdateExpense() returns invalid request body", func(t *testing.T) {
		body := `
			{
				"amount": "79",
				"title": "strawberry smoothie",
				"note": "night market promotion discount 10 bath",
				"tags": ""
			}
		`
		req := httptest.NewRequest(http.MethodPost, "/expenses/:id", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		want := `{"code":400,"message":"invalid request body"}`

		err = h.UpdateExpense(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})

	t.Run("UpdateExpense() returns mot found", func(t *testing.T) {
		exp := expense.Expense{
			ID:     1,
			Amount: 75,
			Title:  "Halo Kitty",
			Note:   "buy tea",
			Tags:   []string{"drinks", "juices"},
		}

		mock.ExpectQuery("SELECT (.+) FROM expenses").WithArgs(exp.ID).WillReturnError(expense.ErrNotFound)

		byt, _ := json.Marshal(exp)
		req := httptest.NewRequest(http.MethodPost, "/expenses/:id", strings.NewReader(string(byt)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		want := `{"code":404,"message":"not found"}`

		err = h.UpdateExpense(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})
}

func TestHandlerGetExpenseByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []string{"id", "amount", "title", "note", "tags"}
	e := echo.New()
	svc := expense.NewService(db)
	h := &handler{svc}

	t.Run("GetExpenseByID()", func(t *testing.T) {
		exp := expense.Expense{
			ID:     2,
			Amount: 105,
			Title:  "strawberry",
			Note:   "night",
			Tags:   []string{"food", "beverage"},
		}

		rows := sqlmock.NewRows(columns).AddRow(exp.ID, exp.Amount, exp.Title, exp.Note, pq.Array(exp.Tags))
		mock.ExpectQuery("SELECT (.+) FROM expenses").WithArgs(exp.ID).WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/expenses/:id", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprintf("%d", exp.ID))
		want := `{"id":2,"amount":105,"title":"strawberry","note":"night","tags":["food","beverage"]}`

		err = h.GetExpenseByID(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})

	t.Run("GetExpenseByID() returns invalid params", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM expenses").WillReturnError(expense.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/expenses/:id", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("A")
		want := `{"code":400,"message":"invalid params"}`
		err = h.GetExpenseByID(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})

	t.Run("GetExpenseByID() returns not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM expenses").WillReturnError(expense.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/expenses/:id", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		want := `{"code":404,"message":"not found"}`
		err = h.GetExpenseByID(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})
}

func TestHandlerListExpenses(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []string{"id", "amount", "title", "note", "tags"}
	e := echo.New()
	svc := expense.NewService(db)
	h := &handler{svc}

	t.Run("ListExpenses()", func(t *testing.T) {
		exps := []expense.Expense{
			{
				ID:     2,
				Amount: 65,
				Title:  "Ice Milk",
				Note:   "",
				Tags:   []string{"drinks", "juices"},
			},
			{
				ID:     3,
				Amount: 100,
				Title:  "Ice Chocolate",
				Note:   "",
				Tags:   []string{"drinks", "juices"},
			},
		}

		rows := sqlmock.NewRows(columns)
		for _, v := range exps {
			rows = rows.AddRow(v.ID, v.Amount, v.Title, v.Note, pq.Array(v.Tags))
		}
		mock.ExpectQuery("SELECT (.+) FROM expenses").WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		want := `[{"id":2,"amount":65,"title":"Ice Milk","note":"","tags":["drinks","juices"]},{"id":3,"amount":100,"title":"Ice Chocolate","note":"","tags":["drinks","juices"]}]`

		err = h.ListExpenses(c)

		if assert.NoError(t, err) {
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, want, strings.TrimSpace(rec.Body.String()))
		}
	})
}
