package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samsunil1999/url-shortener/internal/service"
)

type Handler struct {
	svc    service.URLService
	logger *slog.Logger
}

func NewHandler(svc service.URLService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Shorten(c *gin.Context) {
	var req service.ShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	url, err := h.svc.Shorten(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAliasConflict):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrAliasInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			h.logger.Error("shorten error", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"short_code":   url.ShortCode,
		"short_url":    c.Request.Host + "/" + url.ShortCode,
		"original_url": url.OriginalURL,
		"expires_at":   url.ExpiresAt,
		"created_at":   url.CreatedAt,
	})
}

func (h *Handler) Redirect(c *gin.Context) {
	code := c.Param("code")

	originalURL, err := h.svc.Resolve(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	go h.svc.RecordClick(code, c.Request)
	c.Redirect(http.StatusMovedPermanently, originalURL)
}

func (h *Handler) GetStats(c *gin.Context) {
	code := c.Param("code")

	stats, err := h.svc.GetStats(c.Request.Context(), code)
	if err != nil {
		h.logger.Error("stats error", "error", err, "code", code)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) Delete(c *gin.Context) {
	code := c.Param("code")

	if err := h.svc.Delete(c.Request.Context(), code); err != nil {
		h.logger.Error("delete error", "error", err, "code", code)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
