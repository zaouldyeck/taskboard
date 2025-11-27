package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	pb "github.com/zaouldyeck/taskboard/api/proto/task/v1"
	"github.com/zaouldyeck/taskboard/internal/gateway/grpcclient"
)

type TaskHandler struct {
	taskClient *grpcclient.TaskClient
}

type CreateTaskRequest struct {
	BoardID     int64  `json:"board_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedBy   int64  `json:"created_by"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type ListTasksResponse struct {
	Tasks      []*pb.Task `json:"tasks"`
	TotalCount int32      `json:"total_count"`
	Page       int32      `json:"page"`
	PageSize   int32      `json:"page_size"`
}

func NewTaskHandler(taskClient *grpcclient.TaskClient) *TaskHandler {
	return &TaskHandler{taskClient: taskClient}
}

// Helper functions.

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func respondWithError(w http.ResponseWriter, code int, message, details string) {
	respondWithJSON(w, code, ErrorResponse{
		Error:   message,
		Message: details,
	})
}

// parseInt64Query parses int64 query parameter and returns value.
func parseInt64Query(r *http.Request, key string, defaultValue int64) int64 {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return val
}

// parseInt32Query parses int32 query parameter and returns value.
func parseInt32Query(r *http.Request, key string, defaultValue int32) int32 {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int32(val)
}

func parseBoolQuery(r *http.Request, key string) *bool {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return nil
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return nil
	}
	return &val
}

// extractTask extracts task ID from URL.
func extractTaskID(r *http.Request) (int64, error) {
	path := r.URL.Path

	// Strip "/api/tasks/" prefix.
	if !strings.HasPrefix(path, "/api/tasks/") {
		return 0, fmt.Errorf("invalid path")
	}
	taskId := strings.TrimPrefix(path, "/api/tasks/")
	if taskId == "" {
		return 0, fmt.Errorf("task ID is required")
	}

	return strconv.ParseInt(taskId, 10, 64)
}

// CreateTask handler for POST to "/api/tasks/".
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if req.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Title is required", "")
		return
	}
	if req.BoardID == 0 {
		respondWithError(w, http.StatusBadRequest, "Board ID is required", "")
		return
	}

	task, err := h.taskClient.CreateTask(r.Context(), &pb.CreateTaskRequest{
		BoardId:     req.BoardID,
		Title:       req.Title,
		Description: req.Description,
		CreatedBy:   req.CreatedBy,
	})
	if err != nil {
		log.Printf("Error creating task: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create task", err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, task)
}

// GetTask handler for GET "/api/tasks/:id".
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskId, err := extractTaskID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID", err.Error())
		return
	}

	task, err := h.taskClient.GetTask(r.Context(), taskId)
	if err != nil {
		log.Printf("Error getting task: %v", err)
		respondWithError(w, http.StatusNotFound, "Task not found", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, task)
}

// ListTasks handles GET "/api/tasks".
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Store query params.
	boardId := parseInt64Query(r, "board_id", 1)
	pageSize := parseInt32Query(r, "page_size", 10)
	pageNumber := parseInt32Query(r, "page", 1)
	completed := parseBoolQuery(r, "completed")

	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 10
	}

	tasks, totalCount, err := h.taskClient.ListTasks(r.Context(), boardId, completed, pageSize, pageNumber)
	if err != nil {
		log.Printf("Error listing tasks: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to list tasks", err.Error())
		return
	}

	response := ListTasksResponse{
		Tasks:      tasks,
		TotalCount: totalCount,
		Page:       pageNumber,
		PageSize:   pageSize,
	}
	respondWithJSON(w, http.StatusOK, response)
}

// UpdateTask handles PUT "/api/tasks/:id".
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskId, err := extractTaskID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID", err.Error())
		return
	}

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	grpcReq := &pb.UpdateTaskRequest{
		Id: taskId,
	}
	if req.Title != nil {
		grpcReq.Title = req.Title
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.Completed != nil {
		grpcReq.Completed = req.Completed
	}

	task, err := h.taskClient.UpdateTask(r.Context(), grpcReq)
	if err != nil {
		log.Printf("Error updating task: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to update task", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, task)
}

// DeleteTask handles DELETE "/api/tasks/:id".
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskId, err := extractTaskID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid task ID", err.Error())
		return
	}

	success, err := h.taskClient.DeleteTask(r.Context(), taskId)
	if err != nil {
		log.Printf("Error deleting task: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to delete task", err.Error())
		return
	}

	if success {
		w.WriteHeader(http.StatusNoContent) // HTTP status code: 204 No Content.
	} else {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete task", "")
	}
}
