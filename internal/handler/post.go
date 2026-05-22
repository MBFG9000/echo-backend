package handler

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Post struct {
	posts domain.PostService
}

type createPostRequest struct {
	Content string `json:"content" binding:"required,max=280"`
}

const maxPostAttachmentSize = 10 * 1024 * 1024

type getPostRequest struct {
	ID string `json:"id" binding:"required"`
}

type deletePostRequest struct {
	ID string `json:"id" binding:"required"`
}

type searchPostsRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

func NewPost(posts domain.PostService) *Post {
	return &Post{posts: posts}
}

func (p *Post) RegisterPublic(rg *gin.RouterGroup) {
	rg.POST("/get", p.getByID)
	rg.GET("/attachments/:id", p.getAttachment)
	rg.POST("/search", p.search)
}

func (p *Post) RegisterPrivate(rg *gin.RouterGroup, createMiddleware ...gin.HandlerFunc) {
	if len(createMiddleware) == 0 {
		rg.POST("", p.create)
	} else {
		handlers := make([]gin.HandlerFunc, 0, len(createMiddleware)+1)
		handlers = append(handlers, createMiddleware...)
		handlers = append(handlers, p.create)
		rg.POST("", handlers...)
	}

	rg.POST("/delete", p.delete)
}

func (p *Post) create(c *gin.Context) {
	content, attachment, ok := p.parseCreateRequest(c)
	if !ok {
		return
	}

	userIDValue, ok := c.Get("userID")
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	pseudonymValue, ok := c.Get("pseudonym")
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	pseudonym, ok := pseudonymValue.(string)
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	post, err := p.posts.Create(c.Request.Context(), userID, pseudonym, content, attachment)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, post)
}

func (p *Post) parseCreateRequest(c *gin.Context) (string, *domain.PostAttachment, bool) {
	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPostAttachmentSize+1024*1024)
		if err := c.Request.ParseMultipartForm(maxPostAttachmentSize); err != nil {
			writeDomainError(c, domain.ErrInvalidInput)
			return "", nil, false
		}

		attachment, err := readPostAttachment(c)
		if err != nil {
			writeDomainError(c, domain.ErrInvalidInput)
			return "", nil, false
		}

		return c.PostForm("content"), attachment, true
	}

	var req createPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return "", nil, false
	}

	return req.Content, nil, true
}

func readPostAttachment(c *gin.Context) (*domain.PostAttachment, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, nil
		}
		return nil, err
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxPostAttachmentSize {
		return nil, domain.ErrInvalidInput
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxPostAttachmentSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxPostAttachmentSize {
		return nil, domain.ErrInvalidInput
	}

	fileName := strings.TrimSpace(filepath.Base(fileHeader.Filename))
	if fileName == "" || fileName == "." {
		fileName = "attachment"
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return &domain.PostAttachment{
		ID:          uuid.New(),
		FileName:    fileName,
		ContentType: contentType,
		Size:        int64(len(data)),
		Data:        data,
	}, nil
}

func (p *Post) delete(c *gin.Context) {
	var req deletePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	userIDValue, ok := c.Get("userID")
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	postID, err := uuid.Parse(req.ID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	err = p.posts.Delete(c.Request.Context(), postID, userID)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (p *Post) getByID(c *gin.Context) {
	var req getPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	postID, err := uuid.Parse(req.ID)
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	post, err := p.posts.GetByID(c.Request.Context(), postID)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, post)
}

func (p *Post) getAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeDomainError(c, domain.ErrInvalidInput)
		return
	}

	attachment, err := p.posts.GetAttachment(c.Request.Context(), attachmentID)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	disposition := mime.FormatMediaType("inline", map[string]string{"filename": attachment.FileName})
	c.Header("Content-Disposition", disposition)
	c.Data(http.StatusOK, attachment.ContentType, attachment.Data)
}

func (p *Post) search(c *gin.Context) {
	var req searchPostsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	posts, err := p.posts.Search(c.Request.Context(), req.Query, req.Limit)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts})
}
