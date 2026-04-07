package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
)

var (
	ErrTeamNotFound        = errors.New("team not found")
	ErrSlugConflict        = errors.New("team slug already exists")
	ErrAlreadyMember       = errors.New("user is already a team member")
	ErrLastAdmin           = errors.New("cannot remove or demote the last admin")
	ErrAlreadyRequested    = errors.New("join request already exists")
	ErrRequestNotFound     = errors.New("join request not found")
	ErrRepoAlreadyAssigned = errors.New("repository already assigned to team")
	ErrRepoNotAssigned     = errors.New("repository not assigned to this team")
	ErrInvalidRole         = errors.New("invalid role")
	ErrSelfRoleChange      = errors.New("cannot change own role")
	ErrTeamRepoNotFound    = errors.New("repository not found for team assignment")
)

type TeamService struct {
	teamRepo     repository.TeamRepository
	memberRepo   repository.TeamMemberRepository
	joinReqRepo  repository.TeamJoinRequestRepository
	teamRepoRepo repository.TeamRepoAssignmentRepository
	repoRepo     repository.RepositoryRepository
	userRepo     repository.UserRepository
	notifService *NotificationService
	log          zerolog.Logger
}

func NewTeamService(
	teamRepo repository.TeamRepository,
	memberRepo repository.TeamMemberRepository,
	joinReqRepo repository.TeamJoinRequestRepository,
	teamRepoRepo repository.TeamRepoAssignmentRepository,
	repoRepo repository.RepositoryRepository,
	userRepo repository.UserRepository,
	notifService *NotificationService,
	log zerolog.Logger,
) *TeamService {
	return &TeamService{
		teamRepo:     teamRepo,
		memberRepo:   memberRepo,
		joinReqRepo:  joinReqRepo,
		teamRepoRepo: teamRepoRepo,
		repoRepo:     repoRepo,
		userRepo:     userRepo,
		notifService: notifService,
		log:          log,
	}
}

// CreateTeam creates a team and adds the creator as admin.
func (s *TeamService) CreateTeam(ctx context.Context, userID uuid.UUID, req *dto.CreateTeamRequest) (*dto.TeamResponse, error) {
	// Check slug uniqueness
	if _, err := s.teamRepo.GetBySlug(ctx, req.Slug); err == nil {
		return nil, ErrSlugConflict
	}

	team := &model.Team{
		Name:      req.Name,
		Slug:      req.Slug,
		CreatedBy: &userID,
	}
	if err := s.teamRepo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}

	// Auto-add creator as admin
	member := &model.TeamMember{
		TeamID: team.ID,
		UserID: userID,
		Role:   model.TeamMemberRoleAdmin,
		Status: model.TeamMemberStatusApproved,
	}
	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("add creator as admin: %w", err)
	}

	resp := dto.NewTeamResponse(team)
	resp.MyRole = string(model.TeamMemberRoleAdmin)
	resp.MemberCount = 1
	return &resp, nil
}

// UpdateTeam updates a team's name and slug.
func (s *TeamService) UpdateTeam(ctx context.Context, teamID uuid.UUID, req *dto.UpdateTeamRequest, newSlug string) (*dto.TeamResponse, error) {
	// Check slug uniqueness (exclude self)
	if existing, err := s.teamRepo.GetBySlug(ctx, newSlug); err == nil && existing.ID != teamID {
		return nil, ErrSlugConflict
	}

	if err := s.teamRepo.Update(ctx, teamID, req.Name, newSlug); err != nil {
		return nil, fmt.Errorf("update team: %w", err)
	}

	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("get updated team: %w", err)
	}

	resp := dto.NewTeamResponse(team)
	return &resp, nil
}

// GetTeamDetail returns team with members and repos.
func (s *TeamService) GetTeamDetail(ctx context.Context, teamID, userID uuid.UUID) (*dto.TeamDetailResponse, error) {
	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTeamNotFound
		}
		return nil, fmt.Errorf("get team: %w", err)
	}

	members, err := s.memberRepo.ListMembersWithUser(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	repos, err := s.teamRepoRepo.ListByTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	teamResp := dto.NewTeamResponse(team)

	// Set caller's role
	for _, m := range members {
		if m.UserID == userID && m.Status == model.TeamMemberStatusApproved {
			teamResp.MyRole = string(m.Role)
			break
		}
	}

	// Count approved members
	approvedCount := 0
	for _, m := range members {
		if m.Status == model.TeamMemberStatusApproved {
			approvedCount++
		}
	}
	teamResp.MemberCount = approvedCount
	teamResp.RepoCount = len(repos)

	memberResps := make([]dto.MemberResponse, len(members))
	for i, m := range members {
		var invitedBy *string
		if m.InvitedBy != nil {
			s := m.InvitedBy.String()
			invitedBy = &s
		}
		memberResps[i] = dto.MemberResponse{
			ID:        m.ID.String(),
			UserID:    m.UserID.String(),
			Username:  m.Username,
			Email:     m.Email,
			AvatarURL: m.AvatarURL,
			Role:      string(m.Role),
			Status:    string(m.Status),
			InvitedBy: invitedBy,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		}
	}

	repoResps := make([]dto.TeamRepoResponse, len(repos))
	for i, r := range repos {
		var addedBy *string
		if r.AddedBy != nil {
			s := r.AddedBy.String()
			addedBy = &s
		}
		repoResps[i] = dto.TeamRepoResponse{
			ID:             r.ID.String(),
			TeamID:         r.TeamID.String(),
			RepositoryID:   r.RepositoryID.String(),
			RepositoryName: r.RepositoryName,
			GithubFullName: r.GithubFullName,
			IsActive:       r.IsActive,
			AddedBy:        addedBy,
			CreatedAt:      r.CreatedAt.Format(time.RFC3339),
		}
	}

	return &dto.TeamDetailResponse{
		TeamResponse: teamResp,
		Members:      memberResps,
		Repositories: repoResps,
	}, nil
}

// ListMyTeams returns teams the user is an approved member of.
func (s *TeamService) ListMyTeams(ctx context.Context, userID uuid.UUID) ([]dto.TeamResponse, error) {
	teams, err := s.memberRepo.ListTeamsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}

	resps := make([]dto.TeamResponse, len(teams))
	for i, t := range teams {
		resps[i] = dto.NewTeamResponse(&t)

		// Get caller's role
		member, err := s.memberRepo.GetByTeamAndUser(ctx, t.ID, userID)
		if err == nil {
			resps[i].MyRole = string(member.Role)
		}

		// Get counts
		memberCount, _ := s.memberRepo.CountMembers(ctx, t.ID)
		resps[i].MemberCount = memberCount

		repoCount, _ := s.teamRepoRepo.CountByTeam(ctx, t.ID)
		resps[i].RepoCount = repoCount
	}
	return resps, nil
}

// ListDiscoverable returns discoverable teams with the user's status.
func (s *TeamService) ListDiscoverable(ctx context.Context, userID uuid.UUID) ([]dto.DiscoverableTeamResponse, error) {
	teams, err := s.teamRepo.ListDiscoverable(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list discoverable: %w", err)
	}

	resps := make([]dto.DiscoverableTeamResponse, len(teams))
	for i, t := range teams {
		resps[i] = dto.DiscoverableTeamResponse{
			ID:          t.ID.String(),
			Name:        t.Name,
			Slug:        t.Slug,
			MemberCount: t.MemberCount,
			RepoCount:   t.RepoCount,
			CreatedAt:   t.CreatedAt.Format(time.RFC3339),
			UserStatus:  t.UserStatus,
		}
	}
	return resps, nil
}

// InviteMember invites a user to a team with a given role.
func (s *TeamService) InviteMember(ctx context.Context, teamID, inviterID, targetUserID uuid.UUID, role string) (*dto.MemberResponse, error) {
	if !isValidRole(role) {
		return nil, ErrInvalidRole
	}

	// Check if already a member
	if _, err := s.memberRepo.GetByTeamAndUser(ctx, teamID, targetUserID); err == nil {
		return nil, ErrAlreadyMember
	}

	member := &model.TeamMember{
		TeamID:    teamID,
		UserID:    targetUserID,
		Role:      model.TeamMemberRole(role),
		Status:    model.TeamMemberStatusPending,
		InvitedBy: &inviterID,
	}
	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, fmt.Errorf("invite member: %w", err)
	}

	inviterStr := inviterID.String()
	return &dto.MemberResponse{
		ID:        member.ID.String(),
		UserID:    targetUserID.String(),
		Role:      role,
		Status:    string(model.TeamMemberStatusPending),
		InvitedBy: &inviterStr,
		CreatedAt: member.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateMember updates a member's role and/or status.
func (s *TeamService) UpdateMember(ctx context.Context, teamID, actorID, targetUserID uuid.UUID, req *dto.UpdateMemberRequest) error {
	member, err := s.memberRepo.GetByTeamAndUser(ctx, teamID, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotTeamMember
		}
		return fmt.Errorf("get member: %w", err)
	}

	// Role change
	if req.Role != "" {
		if actorID == targetUserID {
			return ErrSelfRoleChange
		}
		if !isValidRole(req.Role) {
			return ErrInvalidRole
		}
		// Protect last admin on demotion
		if member.Role == model.TeamMemberRoleAdmin && model.TeamMemberRole(req.Role) != model.TeamMemberRoleAdmin {
			count, err := s.memberRepo.CountAdmins(ctx, teamID)
			if err != nil {
				return fmt.Errorf("count admins: %w", err)
			}
			if count <= 1 {
				return ErrLastAdmin
			}
		}
		if err := s.memberRepo.UpdateRole(ctx, teamID, targetUserID, model.TeamMemberRole(req.Role)); err != nil {
			return fmt.Errorf("update role: %w", err)
		}
	}

	// Status change (approve/reject)
	if req.Status != "" {
		if err := s.memberRepo.UpdateStatus(ctx, teamID, targetUserID, model.TeamMemberStatus(req.Status)); err != nil {
			return fmt.Errorf("update status: %w", err)
		}
	}

	return nil
}

// RemoveMember removes a member from a team. Self-removal is allowed.
func (s *TeamService) RemoveMember(ctx context.Context, teamID, actorID, targetUserID uuid.UUID) error {
	member, err := s.memberRepo.GetByTeamAndUser(ctx, teamID, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotTeamMember
		}
		return fmt.Errorf("get member: %w", err)
	}

	// Protect last admin
	if member.Role == model.TeamMemberRoleAdmin && member.Status == model.TeamMemberStatusApproved {
		count, err := s.memberRepo.CountAdmins(ctx, teamID)
		if err != nil {
			return fmt.Errorf("count admins: %w", err)
		}
		if count <= 1 {
			return ErrLastAdmin
		}
	}

	if err := s.memberRepo.Delete(ctx, teamID, targetUserID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// SubmitJoinRequest creates a join request for a discoverable team.
func (s *TeamService) SubmitJoinRequest(ctx context.Context, teamID, userID uuid.UUID, message string) (*dto.JoinRequestResponse, error) {
	// Check if already a member
	if _, err := s.memberRepo.GetByTeamAndUser(ctx, teamID, userID); err == nil {
		return nil, ErrAlreadyMember
	}

	// Check if already requested
	if existing, err := s.joinReqRepo.GetByTeamAndUser(ctx, teamID, userID); err == nil {
		if existing.Status == model.TeamMemberStatusPending {
			return nil, ErrAlreadyRequested
		}
	}

	var msgPtr *string
	if message != "" {
		msgPtr = &message
	}

	req := &model.TeamJoinRequest{
		TeamID:  teamID,
		UserID:  userID,
		Message: msgPtr,
	}
	if err := s.joinReqRepo.Create(ctx, req); err != nil {
		return nil, fmt.Errorf("create join request: %w", err)
	}

	// Notify team admins and maintainers about the new join request
	go s.notifyTeamAdminsJoinRequest(context.Background(), teamID, userID, req.ID)

	return &dto.JoinRequestResponse{
		ID:        req.ID.String(),
		User:      dto.JoinRequestUserResponse{ID: userID.String()},
		Message:   msgPtr,
		Status:    string(req.Status),
		CreatedAt: req.CreatedAt.Format(time.RFC3339),
		UpdatedAt: req.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// ListJoinRequests returns join requests for a team filtered by status.
func (s *TeamService) ListJoinRequests(ctx context.Context, teamID uuid.UUID, statusFilter string) ([]dto.JoinRequestResponse, error) {
	requests, err := s.joinReqRepo.ListByTeam(ctx, teamID, statusFilter)
	if err != nil {
		return nil, fmt.Errorf("list join requests: %w", err)
	}

	resps := make([]dto.JoinRequestResponse, len(requests))
	for i, r := range requests {
		var reviewedBy *string
		if r.ReviewedBy != nil {
			s := r.ReviewedBy.String()
			reviewedBy = &s
		}
		var roleAssigned *string
		if r.RoleAssigned != nil {
			s := string(*r.RoleAssigned)
			roleAssigned = &s
		}
		resps[i] = dto.JoinRequestResponse{
			ID: r.ID.String(),
			User: dto.JoinRequestUserResponse{
				ID:        r.UserID.String(),
				Username:  r.Username,
				Email:     r.Email,
				AvatarURL: r.AvatarURL,
			},
			Message:      r.Message,
			Status:       string(r.Status),
			RoleAssigned: roleAssigned,
			ReviewedBy:   reviewedBy,
			CreatedAt:    r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
		}
	}
	return resps, nil
}

// ReviewJoinRequest approves or rejects a join request.
func (s *TeamService) ReviewJoinRequest(ctx context.Context, requestID, reviewerID uuid.UUID, status string, role string) error {
	req, err := s.joinReqRepo.GetByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrRequestNotFound
		}
		return fmt.Errorf("get join request: %w", err)
	}

	newStatus := model.TeamMemberStatus(status)
	var roleAssigned *model.TeamMemberRole

	if newStatus == model.TeamMemberStatusApproved {
		if role == "" {
			role = "viewer" // default role on approval
		}
		if !isValidRole(role) {
			return ErrInvalidRole
		}
		r := model.TeamMemberRole(role)
		roleAssigned = &r

		// Create team member
		member := &model.TeamMember{
			TeamID:    req.TeamID,
			UserID:    req.UserID,
			Role:      r,
			Status:    model.TeamMemberStatusApproved,
			InvitedBy: &reviewerID,
		}
		if err := s.memberRepo.Create(ctx, member); err != nil {
			return fmt.Errorf("create member from join request: %w", err)
		}
	}

	if err := s.joinReqRepo.UpdateStatus(ctx, requestID, newStatus, reviewerID, roleAssigned); err != nil {
		return fmt.Errorf("update join request status: %w", err)
	}

	// Notify the requester about the review outcome
	go s.notifyRequesterReviewed(context.Background(), req.TeamID, req.UserID, reviewerID, string(newStatus))

	return nil
}

// AssignRepo assigns a repository to a team.
func (s *TeamService) AssignRepo(ctx context.Context, teamID, repoID, addedBy uuid.UUID) (*dto.TeamRepoResponse, error) {
	// Check repo exists
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTeamRepoNotFound
		}
		return nil, fmt.Errorf("get repo: %w", err)
	}

	// Check not already assigned
	if _, err := s.teamRepoRepo.GetByTeamAndRepo(ctx, teamID, repoID); err == nil {
		return nil, ErrRepoAlreadyAssigned
	}

	tr := &model.TeamRepository{
		TeamID:       teamID,
		RepositoryID: repoID,
		AddedBy:      &addedBy,
	}
	if err := s.teamRepoRepo.Create(ctx, tr); err != nil {
		return nil, fmt.Errorf("assign repo: %w", err)
	}

	addedByStr := addedBy.String()
	return &dto.TeamRepoResponse{
		ID:             tr.ID.String(),
		TeamID:         teamID.String(),
		RepositoryID:   repoID.String(),
		RepositoryName: repo.Name,
		GithubFullName: repo.GithubFullName,
		IsActive:       repo.IsActive,
		AddedBy:        &addedByStr,
		CreatedAt:      tr.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UnassignRepo removes a repository from a team.
func (s *TeamService) UnassignRepo(ctx context.Context, teamID, repoID uuid.UUID) error {
	if _, err := s.teamRepoRepo.GetByTeamAndRepo(ctx, teamID, repoID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrRepoNotAssigned
		}
		return fmt.Errorf("get assignment: %w", err)
	}
	return s.teamRepoRepo.Delete(ctx, teamID, repoID)
}

// DeleteTeam soft-deletes a team.
func (s *TeamService) DeleteTeam(ctx context.Context, teamID uuid.UUID) error {
	return s.teamRepo.SoftDelete(ctx, teamID)
}

func isValidRole(role string) bool {
	switch model.TeamMemberRole(role) {
	case model.TeamMemberRoleAdmin, model.TeamMemberRoleMaintainer, model.TeamMemberRoleViewer:
		return true
	}
	return false
}

// notifyTeamAdminsJoinRequest sends a push notification to all team admins and maintainers
// when a new join request is submitted.
func (s *TeamService) notifyTeamAdminsJoinRequest(ctx context.Context, teamID, requesterID, requestID uuid.UUID) {
	if s.notifService == nil {
		return
	}

	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		s.log.Error().Err(err).Str("team_id", teamID.String()).Msg("failed to get team for join request notification")
		return
	}

	requester, err := s.userRepo.GetByID(ctx, requesterID)
	if err != nil {
		s.log.Error().Err(err).Str("user_id", requesterID.String()).Msg("failed to get requester for join request notification")
		return
	}

	members, err := s.memberRepo.ListMembersByTeam(ctx, teamID)
	if err != nil {
		s.log.Error().Err(err).Str("team_id", teamID.String()).Msg("failed to list members for join request notification")
		return
	}

	actionType := "view_join_requests"
	payload, _ := json.Marshal(map[string]string{
		"team_id":    teamID.String(),
		"team_slug":  team.Slug,
		"request_id": requestID.String(),
	})
	rawPayload := json.RawMessage(payload)

	for _, m := range members {
		if m.Status != model.TeamMemberStatusApproved {
			continue
		}
		if m.Role != model.TeamMemberRoleAdmin && m.Role != model.TeamMemberRoleMaintainer {
			continue
		}

		notif := &model.Notification{
			UserID:        m.UserID,
			Type:          model.NotificationTeamJoinRequest,
			Subject:       fmt.Sprintf("%s wants to join %s", requester.Username, team.Name),
			Body:          fmt.Sprintf("%s has requested to join team %s. Please review the request.", requester.Username, team.Name),
			ActionType:    &actionType,
			ActionPayload: &rawPayload,
			SenderID:      &requesterID,
		}
		if err := s.notifService.CreateAndPublish(ctx, notif); err != nil {
			s.log.Error().Err(err).Str("admin_id", m.UserID.String()).Msg("failed to send join request notification")
		}
	}
}

// notifyRequesterReviewed sends a push notification to the join request author
// when their request is approved or rejected.
func (s *TeamService) notifyRequesterReviewed(ctx context.Context, teamID, requesterID, reviewerID uuid.UUID, status string) {
	if s.notifService == nil {
		return
	}

	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		s.log.Error().Err(err).Str("team_id", teamID.String()).Msg("failed to get team for review notification")
		return
	}

	var subject, body string
	if status == string(model.TeamMemberStatusApproved) {
		subject = fmt.Sprintf("Welcome to %s!", team.Name)
		body = fmt.Sprintf("Your request to join team %s has been approved. You are now a member.", team.Name)
	} else {
		subject = fmt.Sprintf("Join request for %s was declined", team.Name)
		body = fmt.Sprintf("Your request to join team %s has been declined.", team.Name)
	}

	actionType := "view_team"
	payload, _ := json.Marshal(map[string]string{
		"team_id":   teamID.String(),
		"team_slug": team.Slug,
		"status":    status,
	})
	rawPayload := json.RawMessage(payload)

	notif := &model.Notification{
		UserID:        requesterID,
		Type:          model.NotificationTeamJoinRequest,
		Subject:       subject,
		Body:          body,
		ActionType:    &actionType,
		ActionPayload: &rawPayload,
		SenderID:      &reviewerID,
	}
	if err := s.notifService.CreateAndPublish(ctx, notif); err != nil {
		s.log.Error().Err(err).Str("user_id", requesterID.String()).Msg("failed to send review notification")
	}
}
