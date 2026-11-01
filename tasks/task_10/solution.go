package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}

type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	lk   sync.Mutex
	clk  Clock
	next uint64
	bag  map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clk: clock,
		bag: make(map[string]Task),
	}
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}

	r.lk.Lock()
	defer r.lk.Unlock()

	r.next++
	id := strconv.FormatUint(r.next, 10)
	t := Task{
		ID:        id,
		Title:     title,
		Done:      false,
		UpdatedAt: r.clk.Now(),
	}
	r.bag[id] = t
	return t, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.lk.Lock()
	defer r.lk.Unlock()
	t, ok := r.bag[id]
	return t, ok
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.lk.Lock()
	defer r.lk.Unlock()

	out := make([]Task, 0)
	for _, t := range r.bag {
		if t.Done == done {
			out = append(out, t)
		}
	}
	return out
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.lk.Lock()
	defer r.lk.Unlock()

	t, ok := r.bag[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	if t.Done == done {
		return t, nil
	}
	t.Done = done
	t.UpdatedAt = r.clk.Now()
	r.bag[id] = t
	return t, nil
}

type httpHandler struct{ store TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{store: repo}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path

	if p == "/tasks" {
		if r.Method == http.MethodPost {
			h.handleCreate(w, r)
			return
		}
		if r.Method == http.MethodGet {
			h.handleList(w, r)
			return
		}
		http.Error(w, "no", http.StatusMethodNotAllowed)
		return
	}

	if !strings.HasPrefix(p, "/tasks/") {
		http.NotFound(w, r)
		return
	}
	rest := p[len("/tasks/"):]
	if rest == "" || strings.Contains(rest, "/") {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		h.handleGet(w, r, rest)
		return
	}
	if r.Method == http.MethodPatch {
		h.handlePatch(w, r, rest)
		return
	}
	http.Error(w, "no", http.StatusMethodNotAllowed)
}

type postBody struct {
	Title *string `json:"title"`
}

type patchBody struct {
	Done *bool `json:"done"`
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body postBody
	if err := decodeStrictJSON(r.Body, &body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if body.Title == nil {
		http.Error(w, "no title", http.StatusBadRequest)
		return
	}

	t, err := h.store.Create(*body.Title)
	if errors.Is(err, ErrInvalidTitle) {
		http.Error(w, "title", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	t, ok := h.store.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()["done"]
	wantDone := false
	if len(vals) == 0 {
		wantDone = false
	} else if len(vals) != 1 || (vals[0] != "true" && vals[0] != "false") {
		http.Error(w, "bad done", http.StatusBadRequest)
		return
	} else {
		wantDone = vals[0] == "true"
	}

	list := h.store.List(wantDone)
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.ID < b.ID
		}
		return a.UpdatedAt.After(b.UpdatedAt)
	})
	writeJSON(w, http.StatusOK, list)
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body patchBody
	if err := decodeStrictJSON(r.Body, &body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if body.Done == nil {
		http.Error(w, "no done", http.StatusBadRequest)
		return
	}

	t, err := h.store.SetDone(id, *body.Done)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "fail", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func decodeStrictJSON(r io.Reader, v any) error {
	d := json.NewDecoder(r)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	err := d.Decode(&struct{}{})
	if err == nil {
		return errors.New("extra stuff")
	}
	if err != io.EOF {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
