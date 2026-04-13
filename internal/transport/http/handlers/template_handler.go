package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	tpldomain "example.com/taskservice/internal/domain/tasktemplate"
	tplusecase "example.com/taskservice/internal/usecase/tasktemplate"
)

type TemplateHandler struct {
	usecase tplusecase.Usecase
}

func NewTemplateHandler(usecase tplusecase.Usecase) *TemplateHandler {
	return &TemplateHandler{usecase: usecase}
}

func (h *TemplateHandler) CreateAndGenerate(w http.ResponseWriter, r *http.Request) {
	var req createTemplateWithGenerateDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	input, err := createInputFromCreateTemplateDTO(req.createTemplateDTO)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	from, err := time.Parse(time.DateOnly, req.GenerateFrom)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid generate_from format, expected YYYY-MM-DD"))
		return
	}

	to, err := time.Parse(time.DateOnly, req.GenerateTo)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid generate_to format, expected YYYY-MM-DD"))
		return
	}

	if from.After(to) {
		writeError(w, http.StatusBadRequest, errors.New("generate_from must not be after generate_to"))
		return
	}

	created, gen, err := h.usecase.CreateWithGeneration(r.Context(), input, from, to)
	if err != nil {
		writeTemplateError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, createTemplateWithGenerateResponseDTO{
		Template: newTemplateDTO(created, created.RawPeriodicityParams),
		Generation: generateResponseDTO{
			TotalDates:     gen.TotalDates,
			GeneratedCount: gen.GeneratedCount,
			SkippedCount:   int64(gen.TotalDates) - gen.GeneratedCount,
		},
	})
}

func createInputFromCreateTemplateDTO(req createTemplateDTO) (tplusecase.CreateInput, error) {
	startDate, err := time.Parse(time.DateOnly, req.StartDate)
	if err != nil {
		return tplusecase.CreateInput{}, errors.New("invalid start_date format, expected YYYY-MM-DD")
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse(time.DateOnly, *req.EndDate)
		if err != nil {
			return tplusecase.CreateInput{}, errors.New("invalid end_date format, expected YYYY-MM-DD")
		}
		endDate = &parsed
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	return tplusecase.CreateInput{
		Title:             req.Title,
		Description:       req.Description,
		PeriodicityType:   req.PeriodicityType,
		PeriodicityParams: req.PeriodicityParams,
		StartDate:         startDate,
		EndDate:           endDate,
		IsActive:          isActive,
	}, nil
}

func (h *TemplateHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := templateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tpl, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeTemplateError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTemplateDTO(tpl, tpl.RawPeriodicityParams))
}

func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := templateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req updateTemplateDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	startDate, err := time.Parse(time.DateOnly, req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid start_date format, expected YYYY-MM-DD"))
		return
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse(time.DateOnly, *req.EndDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid end_date format, expected YYYY-MM-DD"))
			return
		}
		endDate = &parsed
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	updated, err := h.usecase.Update(r.Context(), id, tplusecase.UpdateInput{
		Title:             req.Title,
		Description:       req.Description,
		PeriodicityType:   req.PeriodicityType,
		PeriodicityParams: req.PeriodicityParams,
		StartDate:         startDate,
		EndDate:           endDate,
		IsActive:          isActive,
	})
	if err != nil {
		writeTemplateError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTemplateDTO(updated, updated.RawPeriodicityParams))
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := templateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeTemplateError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := tplusecase.ListFilter{}
	if raw := r.URL.Query().Get("active_only"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid active_only, expected true or false"))
			return
		}
		filter.ActiveOnly = v
	}

	templates, err := h.usecase.List(r.Context(), filter)
	if err != nil {
		writeTemplateError(w, err)
		return
	}

	response := make([]templateDTO, 0, len(templates))
	for i := range templates {
		response = append(response, newTemplateDTO(&templates[i], templates[i].RawPeriodicityParams))
	}

	writeJSON(w, http.StatusOK, response)
}

func templateIDFromRequest(r *http.Request) (int64, error) {
	rawID := mux.Vars(r)["id"]
	if rawID == "" {
		return 0, errors.New("missing template id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid template id")
	}

	if id <= 0 {
		return 0, errors.New("invalid template id")
	}

	return id, nil
}

func writeTemplateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tpldomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, tpldomain.ErrInvalidInput),
		errors.Is(err, tplusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}
