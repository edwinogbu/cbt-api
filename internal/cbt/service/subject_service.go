package service

import (
	"cbt-api/internal/cbt/dto"
	"cbt-api/internal/cbt/repository"
	"cbt-api/internal/models"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubjectService struct {
	repo *repository.SubjectRepository
	db   *gorm.DB
}

func NewSubjectService(repo *repository.SubjectRepository, db *gorm.DB) *SubjectService {
	return &SubjectService{repo: repo, db: db}
}

func (s *SubjectService) CreateSubject(req *dto.CreateSubjectRequest) (*dto.SubjectResponse, error) {
	// check if code already exists (global uniqueness)
	if existing, _ := s.repo.FindByCode(req.Code); existing != nil {
		return nil, errors.New("subject code already exists")
	}
	subject := &models.Subject{
		ID:          uuid.New(),
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		IsActive:    true,
	}
	if err := s.repo.Create(subject); err != nil {
		return nil, err
	}
	return s.toResponse(subject), nil
}

func (s *SubjectService) GetSubject(id string) (*dto.SubjectResponse, error) {
	subjectID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid subject ID")
	}
	subject, err := s.repo.FindByID(subjectID)
	if err != nil {
		return nil, errors.New("subject not found")
	}
	return s.toResponse(subject), nil
}

func (s *SubjectService) UpdateSubject(id string, req *dto.UpdateSubjectRequest) (*dto.SubjectResponse, error) {
	subjectID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid subject ID")
	}
	subject, err := s.repo.FindByID(subjectID)
	if err != nil {
		return nil, errors.New("subject not found")
	}
	if req.Name != nil {
		subject.Name = *req.Name
	}
	if req.Code != nil {
		// check if new code is unique (if changed)
		if existing, _ := s.repo.FindByCode(*req.Code); existing != nil && existing.ID != subjectID {
			return nil, errors.New("subject code already exists")
		}
		subject.Code = *req.Code
	}
	if req.Description != nil {
		subject.Description = *req.Description
	}
	if req.IsActive != nil {
		subject.IsActive = *req.IsActive
	}
	if err := s.repo.Update(subject); err != nil {
		return nil, err
	}
	return s.toResponse(subject), nil
}

func (s *SubjectService) DeleteSubject(id string) error {
	subjectID, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid subject ID")
	}
	return s.repo.Delete(subjectID)
}

func (s *SubjectService) ListSubjects(page, limit int) ([]dto.SubjectResponse, int64, error) {
	subjects, total, err := s.repo.List(page, limit)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]dto.SubjectResponse, len(subjects))
	for i, sub := range subjects {
		resp[i] = *s.toResponse(&sub)
	}
	return resp, total, nil
}

func (s *SubjectService) ListActiveSubjects() ([]dto.SubjectResponse, error) {
	subjects, err := s.repo.ListActive()
	if err != nil {
		return nil, err
	}
	resp := make([]dto.SubjectResponse, len(subjects))
	for i, sub := range subjects {
		resp[i] = *s.toResponse(&sub)
	}
	return resp, nil
}

func (s *SubjectService) toResponse(subject *models.Subject) *dto.SubjectResponse {
	return &dto.SubjectResponse{
		ID:          subject.ID.String(),
		Name:        subject.Name,
		Code:        subject.Code,
		Description: subject.Description,
		IsActive:    subject.IsActive,
		CreatedAt:   subject.CreatedAt,
		UpdatedAt:   subject.UpdatedAt,
	}
}