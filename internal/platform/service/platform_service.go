package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"cbt-api/internal/models"
	"cbt-api/internal/platform/dto"
	"cbt-api/internal/platform/repository"

	"gorm.io/gorm"
)

type PlatformService struct {
	repo *repository.PlatformRepository
}

func NewPlatformService(repo *repository.PlatformRepository) *PlatformService {
	return &PlatformService{repo: repo}
}

// ============================================
// TENANT REGISTRY (Phase 3.3, 3.4, 4)
// ============================================

func (s *PlatformService) ListTenants(q dto.TenantListQuery) (*dto.TenantListResponse, error) {
	page, limit := normalizePaging(q.Page, q.Limit)
	schools, total, err := s.repo.ListSchools(q.Status, q.Search, page, limit)
	if err != nil {
		return nil, err
	}

	resp := &dto.TenantListResponse{Data: make([]dto.TenantSummary, 0, len(schools))}
	for _, sch := range schools {
		summary := dto.TenantSummary{
			ID:              sch.ID,
			Name:            sch.Name,
			Code:            sch.Code,
			ApplicationName: sch.ApplicationName,
			Email:           sch.Email,
			Status:          string(sch.Status),
			IsActive:        sch.IsActive,
			ProvisionedAt:   sch.ProvisionedAt,
			CreatedAt:       sch.CreatedAt,
		}
		if sub, err := s.repo.FindCurrentSubscription(sch.ID); err == nil && sub != nil {
			summary.SubscriptionStatus = string(sub.Status)
			summary.SubscriptionTier = string(sub.Tier)
		}
		resp.Data = append(resp.Data, summary)
	}
	resp.Meta.Page = page
	resp.Meta.Limit = limit
	resp.Meta.Total = total
	return resp, nil
}

func (s *PlatformService) GetTenant(id string) (*dto.TenantDetail, error) {
	sch, err := s.repo.FindSchoolByID(id)
	if err != nil {
		return nil, errors.New("tenant not found")
	}
	detail := &dto.TenantDetail{
		TenantSummary: dto.TenantSummary{
			ID:              sch.ID,
			Name:            sch.Name,
			Code:            sch.Code,
			ApplicationName: sch.ApplicationName,
			Email:           sch.Email,
			Status:          string(sch.Status),
			IsActive:        sch.IsActive,
			ProvisionedAt:   sch.ProvisionedAt,
			CreatedAt:       sch.CreatedAt,
		},
		Address:        sch.Address,
		Phone:          sch.Phone,
		Website:        sch.Website,
		Logo:           sch.Logo,
		PrimaryColor:   sch.PrimaryColor,
		SecondaryColor: sch.SecondaryColor,
	}
	if sub, err := s.repo.FindCurrentSubscription(sch.ID); err == nil && sub != nil {
		detail.SubscriptionStatus = string(sub.Status)
		detail.SubscriptionTier = string(sub.Tier)
	}
	return detail, nil
}

// UpdateBranding is the ONLY write path for a tenant's platform-controlled
// branding fields (Phase 4.2/5.2/5.3) - callers must already be
// authenticated as platform_owner (enforced by the handler's middleware,
// not re-checked here, matching this codebase's existing convention of
// gating authorization at the route/handler layer).
func (s *PlatformService) UpdateBranding(actorUserID, tenantID string, req dto.UpdateBrandingRequest) error {
	sch, err := s.repo.FindSchoolByID(tenantID)
	if err != nil {
		return errors.New("tenant not found")
	}

	before := map[string]string{
		"application_name": sch.ApplicationName,
		"logo":              sch.Logo,
		"primary_color":     sch.PrimaryColor,
		"secondary_color":   sch.SecondaryColor,
	}

	sch.ApplicationName = req.ApplicationName
	if req.Logo != "" {
		sch.Logo = req.Logo
	}
	if req.PrimaryColor != "" {
		sch.PrimaryColor = req.PrimaryColor
	}
	if req.SecondaryColor != "" {
		sch.SecondaryColor = req.SecondaryColor
	}

	if err := s.repo.UpdateSchool(sch); err != nil {
		return err
	}

	after := map[string]string{
		"application_name": sch.ApplicationName,
		"logo":              sch.Logo,
		"primary_color":     sch.PrimaryColor,
		"secondary_color":   sch.SecondaryColor,
	}
	s.writeAudit(actorUserID, "tenant.branding.update", "school", tenantID, map[string]interface{}{
		"before": before,
		"after":  after,
	}, "success")
	return nil
}

// ProvisionTenant activates a tenant (Phase 3.4). Idempotent: calling it
// again on an already-ACTIVE tenant is a harmless no-op rather than a
// duplicate provisioning action, per the plan's explicit requirement.
func (s *PlatformService) ProvisionTenant(actorUserID, tenantID string, req dto.ProvisionTenantRequest) error {
	sch, err := s.repo.FindSchoolByID(tenantID)
	if err != nil {
		return errors.New("tenant not found")
	}

	if sch.Status == models.SchoolStatusActive && sch.ProvisionedAt != nil {
		// Already provisioned - idempotent no-op, but still allow an
		// application-name update to go through the same audited path.
		if req.ApplicationName != "" && req.ApplicationName != sch.ApplicationName {
			sch.ApplicationName = req.ApplicationName
			if err := s.repo.UpdateSchool(sch); err != nil {
				return err
			}
			s.writeAudit(actorUserID, "tenant.provision.rename", "school", tenantID, map[string]interface{}{
				"application_name": req.ApplicationName,
			}, "success")
		}
		return nil
	}

	now := time.Now()
	sch.Status = models.SchoolStatusActive
	sch.IsActive = true
	sch.ProvisionedAt = &now
	sch.ProvisionedBy = &actorUserID
	if req.ApplicationName != "" {
		sch.ApplicationName = req.ApplicationName
	} else if sch.ApplicationName == "" {
		sch.ApplicationName = sch.Name
	}

	if err := s.repo.UpdateSchool(sch); err != nil {
		s.writeAudit(actorUserID, "tenant.provision", "school", tenantID, nil, "failure")
		return err
	}

	s.writeAudit(actorUserID, "tenant.provision", "school", tenantID, map[string]interface{}{
		"application_name": sch.ApplicationName,
		"status":            string(sch.Status),
	}, "success")
	return nil
}

func (s *PlatformService) SuspendTenant(actorUserID, tenantID string, req dto.SuspendTenantRequest) error {
	sch, err := s.repo.FindSchoolByID(tenantID)
	if err != nil {
		return errors.New("tenant not found")
	}

	sch.Status = models.SchoolStatusSuspended
	sch.IsActive = false
	if err := s.repo.UpdateSchool(sch); err != nil {
		s.writeAudit(actorUserID, "tenant.suspend", "school", tenantID, nil, "failure")
		return err
	}

	s.writeAudit(actorUserID, "tenant.suspend", "school", tenantID, map[string]interface{}{
		"reason": req.Reason,
	}, "success")
	return nil
}

// ============================================
// AUDIT LOG (Phase 11)
// ============================================

func (s *PlatformService) ListAuditLogs(q dto.AuditLogQuery) (*dto.AuditLogListResponse, error) {
	page, limit := normalizePaging(q.Page, q.Limit)
	logs, total, err := s.repo.ListAuditLogs(q.Action, page, limit)
	if err != nil {
		return nil, err
	}
	resp := &dto.AuditLogListResponse{Data: make([]dto.AuditLogEntry, 0, len(logs))}
	for _, l := range logs {
		resp.Data = append(resp.Data, dto.AuditLogEntry{
			ID:          l.ID,
			ActorUserID: l.ActorUserID,
			ActorRole:   l.ActorRole,
			Action:      l.Action,
			TargetType:  l.TargetType,
			TargetID:    l.TargetID,
			Details:     l.Details,
			Outcome:     l.Outcome,
			CreatedAt:   l.CreatedAt,
		})
	}
	resp.Meta.Page = page
	resp.Meta.Limit = limit
	resp.Meta.Total = total
	return resp, nil
}

// writeAudit never returns an error to the caller - a failure to write
// an audit row must not block the underlying privileged operation that
// already succeeded (or already failed and is being recorded as such).
// It logs failures to the server's own log instead of silently dropping
// them. actorRole is always "platform_owner" here since every caller of
// this service is already gated to that role by the handler layer.
func (s *PlatformService) writeAudit(actorUserID, action, targetType, targetID string, details interface{}, outcome string) {
	var detailsJSON string
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			detailsJSON = string(b)
		}
	}
	entry := &models.TenantAuditLog{
		ActorUserID: actorUserID,
		ActorRole:   "platform_owner",
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Details:     detailsJSON,
		Outcome:     outcome,
	}
	if err := s.repo.CreateAuditLog(entry); err != nil {
		fmt.Printf("⚠️ [TenantAuditLog] failed to write audit entry for action=%s target=%s: %v\n", action, targetID, err)
	}
}

// ============================================
// LAN SERVER CONFIG (Phase 10)
// ============================================

func (s *PlatformService) GetLANConfig(schoolID string) (*dto.LANConfigResponse, error) {
	cfg, err := s.repo.FindLANConfigBySchool(schoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.LANConfigResponse{SchoolID: schoolID, LastTestStatus: "untested"}, nil
		}
		return nil, err
	}
	return lanConfigToDTO(cfg), nil
}

func (s *PlatformService) UpdateLANConfig(actorUserID, schoolID string, req dto.UpdateLANConfigRequest) (*dto.LANConfigResponse, error) {
	if _, err := s.repo.FindSchoolByID(schoolID); err != nil {
		return nil, errors.New("tenant not found")
	}

	cfg, err := s.repo.FindLANConfigBySchool(schoolID)
	changeType := "update"
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		cfg = &models.LANServerConfig{SchoolID: schoolID}
		changeType = "create"
	}

	cfg.ServerAddress = req.ServerAddress
	cfg.Port = req.Port
	cfg.UpdatedBy = actorUserID
	// A config change invalidates the previous connectivity test result -
	// never claim a configuration is good until it's explicitly retested.
	cfg.LastTestStatus = "untested"
	cfg.LastTestedAt = nil
	cfg.LastTestError = ""

	if err := s.repo.UpsertLANConfig(cfg); err != nil {
		s.writeAudit(actorUserID, "lan.config."+changeType, "lan_config", schoolID, nil, "failure")
		return nil, err
	}

	s.recordLANHistory(cfg, changeType, actorUserID)
	s.writeAudit(actorUserID, "lan.config."+changeType, "lan_config", schoolID, map[string]interface{}{
		"server_address": req.ServerAddress,
		"port":            req.Port,
	}, "success")
	return lanConfigToDTO(cfg), nil
}

func (s *PlatformService) SetLANConfigActive(actorUserID, schoolID string, active bool) (*dto.LANConfigResponse, error) {
	cfg, err := s.repo.FindLANConfigBySchool(schoolID)
	if err != nil {
		return nil, errors.New("no LAN configuration exists for this tenant yet")
	}
	cfg.IsActive = active
	cfg.UpdatedBy = actorUserID
	if err := s.repo.UpsertLANConfig(cfg); err != nil {
		return nil, err
	}
	action := "lan.config.activate"
	if !active {
		action = "lan.config.deactivate"
	}
	s.recordLANHistory(cfg, action, actorUserID)
	s.writeAudit(actorUserID, action, "lan_config", schoolID, nil, "success")
	return lanConfigToDTO(cfg), nil
}

// TestLANConnectivity performs a genuine, bounded TCP dial to the
// configured address/port - it never fabricates a success result. A
// school on a different network than this server (the common case for
// a cloud-hosted platform testing a school's on-premises LAN node) is
// expected to fail here; that's an honest "not reachable from here"
// result, not a bug, and the UI must present it as such rather than as
// a generic error.
func (s *PlatformService) TestLANConnectivity(actorUserID, schoolID string) (*dto.LANConnectivityTestResponse, error) {
	cfg, err := s.repo.FindLANConfigBySchool(schoolID)
	if err != nil {
		return nil, errors.New("no LAN configuration exists for this tenant yet")
	}

	start := time.Now()
	address := net.JoinHostPort(cfg.ServerAddress, strconv.Itoa(cfg.Port))
	conn, dialErr := net.DialTimeout("tcp", address, 5*time.Second)
	elapsed := time.Since(start).Milliseconds()

	now := time.Now()
	cfg.LastTestedAt = &now
	resp := &dto.LANConnectivityTestResponse{ResponseTime: elapsed}
	if dialErr != nil {
		cfg.LastTestStatus = "failed"
		cfg.LastTestError = dialErr.Error()
		resp.Success = false
		resp.Message = "Connection failed: " + dialErr.Error()
	} else {
		conn.Close()
		cfg.LastTestStatus = "success"
		cfg.LastTestError = ""
		resp.Success = true
		resp.Message = "Connection succeeded"
	}

	if err := s.repo.UpsertLANConfig(cfg); err != nil {
		fmt.Printf("⚠️ [LANConfig] failed to persist test result for school=%s: %v\n", schoolID, err)
	}
	s.writeAudit(actorUserID, "lan.config.test", "lan_config", schoolID, map[string]interface{}{
		"success":       resp.Success,
		"response_time": elapsed,
	}, "success")
	return resp, nil
}

func (s *PlatformService) ListLANConfigHistory(schoolID string) ([]dto.LANConfigHistoryEntry, error) {
	history, err := s.repo.ListLANConfigHistory(schoolID, 50)
	if err != nil {
		return nil, err
	}
	out := make([]dto.LANConfigHistoryEntry, 0, len(history))
	for _, h := range history {
		out = append(out, dto.LANConfigHistoryEntry{
			ID:            h.ID,
			ServerAddress: h.ServerAddress,
			Port:          h.Port,
			IsActive:      h.IsActive,
			ChangeType:    h.ChangeType,
			ChangedBy:     h.ChangedBy,
			CreatedAt:     h.CreatedAt,
		})
	}
	return out, nil
}

// RollbackLANConfig restores a school's LAN config to a specific prior
// history entry (Phase 10.2's "roll back to a known valid configuration").
func (s *PlatformService) RollbackLANConfig(actorUserID, schoolID, historyID string) (*dto.LANConfigResponse, error) {
	history, err := s.repo.ListLANConfigHistory(schoolID, 50)
	if err != nil {
		return nil, err
	}
	var target *models.LANServerConfigHistory
	for i := range history {
		if history[i].ID == historyID {
			target = &history[i]
			break
		}
	}
	if target == nil {
		return nil, errors.New("history entry not found for this tenant")
	}

	cfg, err := s.repo.FindLANConfigBySchool(schoolID)
	if err != nil {
		return nil, errors.New("no current LAN configuration to roll back")
	}
	cfg.ServerAddress = target.ServerAddress
	cfg.Port = target.Port
	cfg.IsActive = target.IsActive
	cfg.UpdatedBy = actorUserID
	cfg.LastTestStatus = "untested"
	cfg.LastTestedAt = nil
	cfg.LastTestError = ""

	if err := s.repo.UpsertLANConfig(cfg); err != nil {
		return nil, err
	}
	s.recordLANHistory(cfg, "rollback", actorUserID)
	s.writeAudit(actorUserID, "lan.config.rollback", "lan_config", schoolID, map[string]interface{}{
		"rolled_back_to": historyID,
	}, "success")
	return lanConfigToDTO(cfg), nil
}

func (s *PlatformService) recordLANHistory(cfg *models.LANServerConfig, changeType, actorUserID string) {
	entry := &models.LANServerConfigHistory{
		ConfigID:      cfg.ID,
		SchoolID:      cfg.SchoolID,
		ServerAddress: cfg.ServerAddress,
		Port:          cfg.Port,
		IsActive:      cfg.IsActive,
		ChangeType:    changeType,
		ChangedBy:     actorUserID,
	}
	if err := s.repo.CreateLANConfigHistory(entry); err != nil {
		fmt.Printf("⚠️ [LANConfigHistory] failed to write history entry for school=%s: %v\n", cfg.SchoolID, err)
	}
}

func lanConfigToDTO(cfg *models.LANServerConfig) *dto.LANConfigResponse {
	return &dto.LANConfigResponse{
		ID:             cfg.ID,
		SchoolID:       cfg.SchoolID,
		ServerAddress:  cfg.ServerAddress,
		Port:           cfg.Port,
		IsActive:       cfg.IsActive,
		LastTestedAt:   cfg.LastTestedAt,
		LastTestStatus: cfg.LastTestStatus,
		LastTestError:  cfg.LastTestError,
		UpdatedAt:      cfg.UpdatedAt,
	}
}

func normalizePaging(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}
