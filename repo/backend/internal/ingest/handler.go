package ingest

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eagle-point/service-portal/internal/apierr"
)

// Handler exposes ingest source + job HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ─── Sources ─────────────────────────────────────────────────────────────────

func (h *Handler) ListSources(c *gin.Context) {
	items, err := h.svc.ListSources(c.Request.Context())
	if err != nil {
		apierr.InternalError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sources": items})
}

type sourceRequest struct {
	Name       string `json:"name"        binding:"required"`
	SourceType string `json:"source_type" binding:"required"`
	Config     string `json:"config"`
	IsActive   *bool  `json:"is_active"`
}

func (h *Handler) CreateSource(c *gin.Context) {
	var req sourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, "validation_error", err.Error())
		return
	}
	src, err := h.svc.CreateSource(c.Request.Context(), CreateSourceInput{
		Name: req.Name, SourceType: req.SourceType, Config: req.Config,
	})
	if err != nil {
		if errors.Is(err, ErrValidation) {
			apierr.UnprocessableEntity(c, "validation_error", err.Error(), nil)
			return
		}
		apierr.InternalError(c)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"source": src})
}

func (h *Handler) UpdateSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		apierr.BadRequest(c, "invalid_param", "id must be a positive integer")
		return
	}
	var req sourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, "validation_error", err.Error())
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	src, err := h.svc.UpdateSource(c.Request.Context(), id, CreateSourceInput{
		Name: req.Name, SourceType: req.SourceType, Config: req.Config,
	}, active)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			apierr.NotFound(c, "source")
		case errors.Is(err, ErrValidation):
			apierr.UnprocessableEntity(c, "validation_error", err.Error(), nil)
		default:
			apierr.InternalError(c)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"source": src})
}

// ─── Jobs ────────────────────────────────────────────────────────────────────

type createJobRequest struct {
	SourceID uint64 `json:"source_id" binding:"required"`
}

func (h *Handler) CreateJob(c *gin.Context) {
	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, "validation_error", err.Error())
		return
	}
	j, err := h.svc.CreateJob(c.Request.Context(), req.SourceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.NotFound(c, "source")
			return
		}
		apierr.InternalError(c)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"job": j})
}

func (h *Handler) ListJobs(c *gin.Context) {
	sourceID, _ := strconv.ParseUint(c.Query("source_id"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.svc.ListJobs(c.Request.Context(), sourceID, limit)
	if err != nil {
		apierr.InternalError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"jobs": items})
}

func (h *Handler) GetJob(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		apierr.BadRequest(c, "invalid_param", "id must be a positive integer")
		return
	}
	j, err := h.svc.GetJob(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.NotFound(c, "job")
			return
		}
		apierr.InternalError(c)
		return
	}
	// Include latest checkpoint if any
	cp, _ := h.svc.LoadCheckpoint(c.Request.Context(), j.SourceID, j.ID)
	c.JSON(http.StatusOK, gin.H{"job": j, "checkpoint": cp})
}

// ─── Schema versions ─────────────────────────────────────────────────────────

func (h *Handler) ListSchemaVersions(c *gin.Context) {
	sourceID, err := strconv.ParseUint(c.Param("source_id"), 10, 64)
	if err != nil || sourceID == 0 {
		apierr.BadRequest(c, "invalid_param", "source_id must be a positive integer")
		return
	}
	items, err := h.svc.ListSchemaVersions(c.Request.Context(), sourceID)
	if err != nil {
		apierr.InternalError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"versions": items})
}
