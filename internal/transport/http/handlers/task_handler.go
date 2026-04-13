package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
	taskdomain "example.com/taskservice/internal/domain/task"
	tplusecase "example.com/taskservice/internal/usecase/tasktemplate"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type TaskHandler struct {
	tasks     taskusecase.Usecase
	templates tplusecase.Usecase
}

func NewTaskHandler(tasks taskusecase.Usecase, templates tplusecase.Usecase) *TaskHandler {
	return &TaskHandler{tasks: tasks, templates: templates}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req taskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.tasks.Create(r.Context(), taskusecase.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(created))
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	task, err := h.tasks.GetByID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	expandTpl, err := parseExpandTemplate(r.URL.Query().Get("expand"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if !expandTpl {
		writeJSON(w, http.StatusOK, newTaskDTO(task))
		return
	}

	payload, err := h.taskResponseWithTemplate(r.Context(), task)
	if err != nil {
		writeTemplateExpandError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, payload)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req taskUpdateDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	in := taskusecase.UpdateInput{
		Title:              req.Title,
		Description:        req.Description,
		Status:             req.Status,
		UnlinkTemplate:     req.UnlinkTemplate,
		TemplateID:         req.TemplateID,
		ClearScheduledDate: req.ClearScheduledDate,
	}
	if req.ScheduledDate != nil {
		sd, err := time.Parse(time.DateOnly, *req.ScheduledDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid scheduled_date format, expected YYYY-MM-DD"))
			return
		}
		in.ScheduledDate = &sd
	}

	updated, err := h.tasks.Update(r.Context(), id, in)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(updated))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.tasks.Delete(r.Context(), id); err != nil {
		writeUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	var filter taskusecase.ListFilter
	q := r.URL.Query()

	if raw := q.Get("template_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, errors.New("invalid template_id"))
			return
		}
		filter.TemplateID = &id
	}

	if raw := q.Get("schedule_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, errors.New("invalid schedule_id"))
			return
		}
		filter.TemplateID = &id
	}

	if raw := q.Get("date_from"); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date_from: expected YYYY-MM-DD"))
			return
		}
		filter.DateFrom = &t
	}

	if raw := q.Get("date_to"); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date_to: expected YYYY-MM-DD"))
			return
		}
		filter.DateTo = &t
	}

	if raw := q.Get("date"); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date: expected YYYY-MM-DD"))
			return
		}
		filter.DateFrom = &t
		filter.DateTo = &t
	}

	if raw := q.Get("status"); raw != "" {
		s := taskdomain.Status(raw)
		if !s.Valid() {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid status: %s", raw))
			return
		}
		filter.Status = &s
	}

	expandTpl, err := parseExpandTemplate(q.Get("expand"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tasks, err := h.tasks.List(r.Context(), filter)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	if !expandTpl {
		response := make([]taskDTO, 0, len(tasks))
		for i := range tasks {
			response = append(response, newTaskDTO(&tasks[i]))
		}
		writeJSON(w, http.StatusOK, response)
		return
	}

	tplByID, err := h.loadTemplatesForTasks(r.Context(), tasks)
	if err != nil {
		writeTemplateExpandError(w, err)
		return
	}

	response := make([]taskExpandedDTO, 0, len(tasks))
	for i := range tasks {
		t := &tasks[i]
		var td *templateDTO
		if t.TemplateID != nil {
			if tpl, ok := tplByID[*t.TemplateID]; ok && tpl != nil {
				d := newTemplateDTO(tpl, tpl.RawPeriodicityParams)
				td = &d
			}
		}
		response = append(response, newTaskExpandedDTO(t, td))
	}

	writeJSON(w, http.StatusOK, response)
}

func parseExpandTemplate(raw string) (bool, error) {
	if strings.TrimSpace(raw) == "" {
		return false, nil
	}
	var want bool
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(strings.ToLower(part))
		if p == "" {
			continue
		}
		if p != "template" && p != "schedule" {
			return false, fmt.Errorf("invalid expand: supported values are template, schedule (deprecated)")
		}
		want = true
	}
	return want, nil
}

func (h *TaskHandler) taskResponseWithTemplate(ctx context.Context, task *taskdomain.Task) (any, error) {
	if task.TemplateID == nil {
		return newTaskExpandedDTO(task, nil), nil
	}
	tpl, err := h.templates.GetByID(ctx, *task.TemplateID)
	if err != nil {
		if errors.Is(err, tpldomain.ErrNotFound) {
			return newTaskExpandedDTO(task, nil), nil
		}
		return nil, err
	}
	dto := newTemplateDTO(tpl, tpl.RawPeriodicityParams)
	return newTaskExpandedDTO(task, &dto), nil
}

func (h *TaskHandler) loadTemplatesForTasks(ctx context.Context, tasks []taskdomain.Task) (map[int64]*tpldomain.Template, error) {
	seen := make(map[int64]struct{})
	for i := range tasks {
		if tasks[i].TemplateID != nil {
			seen[*tasks[i].TemplateID] = struct{}{}
		}
	}
	out := make(map[int64]*tpldomain.Template, len(seen))
	for id := range seen {
		tpl, err := h.templates.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, tpldomain.ErrNotFound) {
				out[id] = nil
				continue
			}
			return nil, err
		}
		out[id] = tpl
	}
	return out, nil
}

func writeTemplateExpandError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tpldomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, tplusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func getIDFromRequest(r *http.Request) (int64, error) {
	rawID := mux.Vars(r)["id"]
	if rawID == "" {
		return 0, errors.New("missing task id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid task id")
	}

	if id <= 0 {
		return 0, errors.New("invalid task id")
	}

	return id, nil
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}

func writeUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, taskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, taskusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
