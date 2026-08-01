// internal/academic/service/class_arm_service.go
package service

import (
    "context"
    "errors"
    "fmt"
    // "strings"
    "time"

    "cbt-api/internal/academic/dto"
    "cbt-api/internal/academic/repository"
    "cbt-api/internal/models"
    "github.com/google/uuid"
)

type ClassArmService struct {
    repo *repository.ClassArmRepository
}

func NewClassArmService(repo *repository.ClassArmRepository) *ClassArmService {
    return &ClassArmService{repo: repo}
}

// ============================================
// CORE CRUD OPERATIONS
// ============================================

// Create creates a new class arm
func (s *ClassArmService) Create(req *dto.CreateClassArmRequest) (*dto.ClassArmResponse, error) {
    // Validate unique name per school
    exists, err := s.repo.ExistsByName(req.SchoolID, req.Name)
    if err != nil {
        return nil, fmt.Errorf("failed to check name uniqueness: %w", err)
    }
    if exists {
        return nil, fmt.Errorf("class arm with name '%s' already exists for this school", req.Name)
    }

    // Validate unique code per school (if code provided)
    if req.ArmCode != "" {
        exists, err := s.repo.ExistsByCode(req.SchoolID, req.ArmCode)
        if err != nil {
            return nil, fmt.Errorf("failed to check code uniqueness: %w", err)
        }
        if exists {
            return nil, fmt.Errorf("class arm with code '%s' already exists for this school", req.ArmCode)
        }
    }

    arm := &models.ClassArm{
        ID:         uuid.New().String(),
        SchoolID:   req.SchoolID,
        Name:       req.Name,
        ArmCode:    req.ArmCode,
        Capacity:   req.Capacity,
        RoomNumber: req.RoomNumber,
        SortOrder:  req.SortOrder,
        IsActive:   true,
        CreatedAt:  time.Now(),
        UpdatedAt:  time.Now(),
    }

    if err := s.repo.Create(arm); err != nil {
        return nil, fmt.Errorf("failed to create class arm: %w", err)
    }

    return s.toResponse(arm), nil
}

// GetByID retrieves a class arm by ID
func (s *ClassArmService) GetByID(id string) (*dto.ClassArmResponse, error) {
    if id == "" {
        return nil, errors.New("class arm ID is required")
    }

    arm, err := s.repo.FindByID(id)
    if err != nil {
        return nil, err
    }
    return s.toResponse(arm), nil
}

// GetBySchool retrieves all class arms for a school
func (s *ClassArmService) GetBySchool(schoolID string) ([]dto.ClassArmResponse, error) {
    if schoolID == "" {
        return nil, errors.New("school ID is required")
    }

    arms, err := s.repo.FindBySchool(schoolID)
    if err != nil {
        return nil, fmt.Errorf("failed to get class arms: %w", err)
    }

    responses := make([]dto.ClassArmResponse, len(arms))
    for i, arm := range arms {
        responses[i] = *s.toResponse(&arm)
    }
    return responses, nil
}

// ✅ FIXED: Update uses string types, not pointers
func (s *ClassArmService) Update(id string, req *dto.UpdateClassArmRequest) (*dto.ClassArmResponse, error) {
    if id == "" {
        return nil, errors.New("class arm ID is required")
    }

    // Get existing arm
    arm, err := s.repo.FindByID(id)
    if err != nil {
        return nil, err
    }

    // ✅ Name is a string, check if it's not empty and different
    if req.Name != "" && req.Name != arm.Name {
        // Check name uniqueness if name is being changed
        exists, err := s.repo.ExistsByName(arm.SchoolID, req.Name)
        if err != nil {
            return nil, fmt.Errorf("failed to check name uniqueness: %w", err)
        }
        if exists {
            return nil, fmt.Errorf("class arm with name '%s' already exists for this school", req.Name)
        }
        arm.Name = req.Name
    }

    // ✅ ArmCode is a string, check if it's not empty and different
    if req.ArmCode != "" && req.ArmCode != arm.ArmCode {
        // Check code uniqueness if code is being changed
        exists, err := s.repo.ExistsByCode(arm.SchoolID, req.ArmCode)
        if err != nil {
            return nil, fmt.Errorf("failed to check code uniqueness: %w", err)
        }
        if exists {
            return nil, fmt.Errorf("class arm with code '%s' already exists for this school", req.ArmCode)
        }
        arm.ArmCode = req.ArmCode
    }

    // ✅ RoomNumber is a string
    if req.RoomNumber != "" {
        arm.RoomNumber = req.RoomNumber
    }

    // Update other fields (these are pointers)
    if req.Capacity != nil {
        arm.Capacity = *req.Capacity
    }
    if req.SortOrder != nil {
        arm.SortOrder = *req.SortOrder
    }
    if req.IsActive != nil {
        arm.IsActive = *req.IsActive
    }
    arm.UpdatedAt = time.Now()

    if err := s.repo.Update(arm); err != nil {
        return nil, fmt.Errorf("failed to update class arm: %w", err)
    }

    return s.toResponse(arm), nil
}

// Delete soft-deletes a class arm
func (s *ClassArmService) Delete(id string) error {
    if id == "" {
        return errors.New("class arm ID is required")
    }

    // Check if arm exists
    _, err := s.repo.FindByID(id)
    if err != nil {
        return err
    }

    return s.repo.Delete(id)
}

// ============================================
// LIST & FILTER OPERATIONS
// ============================================

// ListClassArms returns paginated class arms with filters (all optional)
func (s *ClassArmService) ListClassArms(ctx context.Context, req *dto.ListClassArmsRequest) (*dto.ClassArmListResponse, error) {
    // Build filters map (only add if provided)
    filters := make(map[string]interface{})

    if req.SchoolID != "" {
        filters["school_id"] = req.SchoolID
    }
    if req.Search != "" {
        filters["search"] = req.Search
    }
    if req.MinCapacity > 0 {
        filters["min_capacity"] = req.MinCapacity
    }
    if req.MaxCapacity > 0 {
        filters["max_capacity"] = req.MaxCapacity
    }
    if req.IsActive != nil {
        filters["is_active"] = req.IsActive
    }

    // Set defaults
    page := req.Page
    if page < 1 {
        page = 1
    }
    limit := req.Limit
    if limit < 1 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }

    sortBy := req.SortBy
    if sortBy == "" {
        sortBy = "created_at"
    }
    sortOrder := req.SortOrder
    if sortOrder == "" {
        sortOrder = "desc"
    }

    // Get class arms (filters will be empty if not provided)
    arms, total, err := s.repo.ListClassArms(ctx, filters, page, limit, sortBy, sortOrder)
    if err != nil {
        return nil, err
    }

    // Convert to response DTOs
    items := make([]dto.ClassArmResponse, len(arms))
    for i, arm := range arms {
        items[i] = *s.toResponse(&arm)
    }

    // Calculate total pages
    totalPages := int((total + int64(limit) - 1) / int64(limit))
    if totalPages < 1 {
        totalPages = 1
    }

    // Build response
    response := &dto.ClassArmListResponse{
        Items: items,
        Pagination: dto.PaginationMeta{
            Page:       page,
            Limit:      limit,
            Total:      total,
            TotalPages: totalPages,
        },
    }

    return response, nil
}

// ============================================
// STATISTICS OPERATIONS
// ============================================

// GetClassArmStats returns class arm statistics
func (s *ClassArmService) GetClassArmStats(ctx context.Context, schoolID string) (*dto.ClassArmStatsResponse, error) {
    return s.repo.GetClassArmStats(ctx, schoolID)
}

// ============================================
// BULK OPERATIONS
// ============================================

// BulkDeleteClassArms deletes multiple class arms
func (s *ClassArmService) BulkDeleteClassArms(ctx context.Context, ids []string) (int64, error) {
    if len(ids) == 0 {
        return 0, errors.New("no IDs provided")
    }
    if len(ids) > 100 {
        return 0, errors.New("maximum 100 IDs allowed per request")
    }
    return s.repo.BulkDeleteClassArms(ctx, ids)
}

// ============================================
// SEARCH OPERATIONS
// ============================================

// SearchClassArms searches class arms by name or code
func (s *ClassArmService) SearchClassArms(ctx context.Context, query string, limit int) ([]dto.ClassArmResponse, error) {
    if query == "" {
        return nil, errors.New("search query is required")
    }
    if len(query) < 2 {
        return nil, errors.New("search query must be at least 2 characters")
    }
    if limit < 1 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }

    arms, err := s.repo.SearchClassArms(ctx, query, limit)
    if err != nil {
        return nil, err
    }

    responses := make([]dto.ClassArmResponse, len(arms))
    for i, arm := range arms {
        responses[i] = *s.toResponse(&arm)
    }
    return responses, nil
}

// ============================================
// HELPER METHODS
// ============================================

// toResponse converts a model to response DTO
func (s *ClassArmService) toResponse(arm *models.ClassArm) *dto.ClassArmResponse {
    return &dto.ClassArmResponse{
        ID:         arm.ID,
        SchoolID:   arm.SchoolID,
        Name:       arm.Name,
        ArmCode:    arm.ArmCode,
        Capacity:   arm.Capacity,
        RoomNumber: arm.RoomNumber,
        SortOrder:  arm.SortOrder,
        IsActive:   arm.IsActive,
        CreatedAt:  arm.CreatedAt,
        UpdatedAt:  arm.UpdatedAt,
    }
}



// package service

// import (
//     "context"
//     "errors"
//     "time"

//     "cbt-api/internal/academic/dto"
//     "cbt-api/internal/academic/repository"
//     "cbt-api/internal/models"
//     "github.com/google/uuid"
// )

// type ClassArmService struct {
//     repo *repository.ClassArmRepository
// }

// func NewClassArmService(repo *repository.ClassArmRepository) *ClassArmService {
//     return &ClassArmService{repo: repo}
// }

// func (s *ClassArmService) Create(req *dto.CreateClassArmRequest) (*dto.ClassArmResponse, error) {
//     arm := &models.ClassArm{
//         ID:         uuid.New().String(),
//         SchoolID:   req.SchoolID,
//         Name:       req.Name,
//         ArmCode:    req.ArmCode,
//         Capacity:   req.Capacity,
//         RoomNumber: req.RoomNumber,
//         SortOrder:  req.SortOrder,
//         IsActive:   true,
//         CreatedAt:  time.Now(),
//         UpdatedAt:  time.Now(),
//     }
    
//     if err := s.repo.Create(arm); err != nil {
//         return nil, err
//     }
    
//     return s.toResponse(arm), nil
// }

// func (s *ClassArmService) GetByID(id string) (*dto.ClassArmResponse, error) {
//     arm, err := s.repo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("class arm not found")
//     }
//     return s.toResponse(arm), nil
// }

// func (s *ClassArmService) GetBySchool(schoolID string) ([]dto.ClassArmResponse, error) {
//     arms, err := s.repo.FindBySchool(schoolID)
//     if err != nil {
//         return nil, err
//     }
    
//     var responses []dto.ClassArmResponse
//     for _, arm := range arms {
//         responses = append(responses, *s.toResponse(&arm))
//     }
//     return responses, nil
// }

// func (s *ClassArmService) Update(id string, req *dto.UpdateClassArmRequest) (*dto.ClassArmResponse, error) {
//     arm, err := s.repo.FindByID(id)
//     if err != nil {
//         return nil, errors.New("class arm not found")
//     }
    
//     if req.Name != "" {
//         arm.Name = req.Name
//     }
//     if req.ArmCode != "" {
//         arm.ArmCode = req.ArmCode
//     }
//     if req.Capacity != nil {
//         arm.Capacity = *req.Capacity
//     }
//     if req.RoomNumber != "" {
//         arm.RoomNumber = req.RoomNumber
//     }
//     if req.SortOrder != nil {
//         arm.SortOrder = *req.SortOrder
//     }
//     if req.IsActive != nil {
//         arm.IsActive = *req.IsActive
//     }
//     arm.UpdatedAt = time.Now()
    
//     if err := s.repo.Update(arm); err != nil {
//         return nil, err
//     }
    
//     return s.toResponse(arm), nil
// }

// func (s *ClassArmService) Delete(id string) error {
//     return s.repo.Delete(id)
// }

// func (s *ClassArmService) toResponse(arm *models.ClassArm) *dto.ClassArmResponse {
//     return &dto.ClassArmResponse{
//         ID:         arm.ID,
//         SchoolID:   arm.SchoolID,
//         Name:       arm.Name,
//         ArmCode:    arm.ArmCode,
//         Capacity:   arm.Capacity,
//         RoomNumber: arm.RoomNumber,
//         SortOrder:  arm.SortOrder,
//         IsActive:   arm.IsActive,
//         CreatedAt:  arm.CreatedAt,
//         UpdatedAt:  arm.UpdatedAt,
//     }
// }


// // ListClassArms returns paginated class arms with filters
// // func (s *ClassArmService) ListClassArms(ctx context.Context, req *dto.ListClassArmsRequest) (*dto.ClassArmListResponse, error) {
// //     // Build filters map
// //     filters := make(map[string]interface{})

// //     if req.SchoolID != "" {
// //         filters["school_id"] = req.SchoolID
// //     }
// //     if req.Search != "" {
// //         filters["search"] = req.Search
// //     }
// //     if req.MinCapacity > 0 {
// //         filters["min_capacity"] = req.MinCapacity
// //     }
// //     if req.MaxCapacity > 0 {
// //         filters["max_capacity"] = req.MaxCapacity
// //     }
// //     if req.IsActive != nil {
// //         filters["is_active"] = req.IsActive
// //     }

// //     // Set defaults
// //     page := req.Page
// //     if page < 1 {
// //         page = 1
// //     }
// //     limit := req.Limit
// //     if limit < 1 {
// //         limit = 20
// //     }
// //     if limit > 100 {
// //         limit = 100
// //     }

// //     sortBy := req.SortBy
// //     if sortBy == "" {
// //         sortBy = "created_at"
// //     }
// //     sortOrder := req.SortOrder
// //     if sortOrder == "" {
// //         sortOrder = "desc"
// //     }

// //     // Get class arms
// //     arms, total, err := s.repo.ListClassArms(ctx, filters, page, limit, sortBy, sortOrder)
// //     if err != nil {
// //         return nil, err
// //     }

// //     // Convert to response DTOs
// //     items := make([]dto.ClassArmResponse, len(arms))
// //     for i, arm := range arms {
// //         items[i] = *s.toResponse(&arm)
// //     }

// //     // Calculate total pages
// //     totalPages := int((total + int64(limit) - 1) / int64(limit))
// //     if totalPages < 1 {
// //         totalPages = 1
// //     }

// //     // Build response
// //     response := &dto.ClassArmListResponse{
// //         Items: items,
// //         Pagination: dto.PaginationMeta{
// //             Page:       page,
// //             Limit:      limit,
// //             Total:      total,
// //             TotalPages: totalPages,
// //         },
// //     }

// //     return response, nil
// // }
// // internal/academic/service/class_arm_service.go

// // ListClassArms returns paginated class arms with filters (all optional)
// func (s *ClassArmService) ListClassArms(ctx context.Context, req *dto.ListClassArmsRequest) (*dto.ClassArmListResponse, error) {
//     // Build filters map (only add if provided)
//     filters := make(map[string]interface{})

//     if req.SchoolID != "" {
//         filters["school_id"] = req.SchoolID
//     }
//     if req.Search != "" {
//         filters["search"] = req.Search
//     }
//     if req.MinCapacity > 0 {
//         filters["min_capacity"] = req.MinCapacity
//     }
//     if req.MaxCapacity > 0 {
//         filters["max_capacity"] = req.MaxCapacity
//     }
//     if req.IsActive != nil {
//         filters["is_active"] = req.IsActive
//     }

//     // Set defaults
//     page := req.Page
//     if page < 1 {
//         page = 1
//     }
//     limit := req.Limit
//     if limit < 1 {
//         limit = 20
//     }
//     if limit > 100 {
//         limit = 100
//     }

//     sortBy := req.SortBy
//     if sortBy == "" {
//         sortBy = "created_at"
//     }
//     sortOrder := req.SortOrder
//     if sortOrder == "" {
//         sortOrder = "desc"
//     }

//     // Get class arms (filters will be empty if not provided)
//     arms, total, err := s.repo.ListClassArms(ctx, filters, page, limit, sortBy, sortOrder)
//     if err != nil {
//         return nil, err
//     }

//     // Convert to response DTOs
//     items := make([]dto.ClassArmResponse, len(arms))
//     for i, arm := range arms {
//         items[i] = *s.toResponse(&arm)
//     }

//     // Calculate total pages
//     totalPages := int((total + int64(limit) - 1) / int64(limit))
//     if totalPages < 1 {
//         totalPages = 1
//     }

//     // Build response
//     response := &dto.ClassArmListResponse{
//         Items: items,
//         Pagination: dto.PaginationMeta{
//             Page:       page,
//             Limit:      limit,
//             Total:      total,
//             TotalPages: totalPages,
//         },
//     }

//     return response, nil
// }

// // GetClassArmStats returns class arm statistics
// func (s *ClassArmService) GetClassArmStats(ctx context.Context, schoolID string) (*dto.ClassArmStatsResponse, error) {
//     return s.repo.GetClassArmStats(ctx, schoolID)
// }

// // BulkDeleteClassArms deletes multiple class arms
// func (s *ClassArmService) BulkDeleteClassArms(ctx context.Context, ids []string) (int64, error) {
//     if len(ids) == 0 {
//         return 0, errors.New("no IDs provided")
//     }
//     return s.repo.BulkDeleteClassArms(ctx, ids)
// }

// // SearchClassArms searches class arms by name or code
// func (s *ClassArmService) SearchClassArms(ctx context.Context, query string, limit int) ([]dto.ClassArmResponse, error) {
//     if query == "" {
//         return nil, errors.New("search query is required")
//     }
//     if limit < 1 {
//         limit = 20
//     }
//     if limit > 100 {
//         limit = 100
//     }

//     arms, err := s.repo.SearchClassArms(ctx, query, limit)
//     if err != nil {
//         return nil, err
//     }

//     responses := make([]dto.ClassArmResponse, len(arms))
//     for i, arm := range arms {
//         responses[i] = *s.toResponse(&arm)
//     }
//     return responses, nil
// }