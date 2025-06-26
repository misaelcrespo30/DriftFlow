package handler

import (
	"context"
	"matters-service/internal/models"
	"time"

	"matters-service/internal/service"
	pb "matters-service/proto"
)

type GRPCServer struct {
	pb.UnimplementedMatterServiceServer
	pb.UnimplementedMatterActivityServiceServer
	pb.UnimplementedMatterRelatedServiceServer
	pb.UnimplementedMatterStatusServiceServer
	pb.UnimplementedMatterActivityCategoryServiceServer
	svc         service.MatterService
	activitySvc service.MatterActivityService
	relatedSvc  service.MatterRelatedService
	statusSvc   service.MatterStatusService
	categorySvc service.MatterActivityCategoryService
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func toProto(m models.Matter) *pb.Matter {
	return &pb.Matter{
		Id:                        int64(m.ID),
		Description:               m.Description,
		OpenDate:                  m.OpenDate.Format(time.RFC3339),
		CloseDate:                 m.CloseDate.Format(time.RFC3339),
		PendingDate:               m.PendingDate.Format(time.RFC3339),
		LimitationDate:            m.LimitationDate.Format(time.RFC3339),
		IsBillable:                m.IsBillable,
		IsLimitationDateSatisfied: m.IsLimitationDateSatisfied,
		StatusId:                  int64(m.StatusID),
		Rate:                      m.Rate,
		PracticeAreaId:            int64(m.PracticeAreaID),
		ClientId:                  int64(m.ClientID),
		OriginatingAttorneyId:     int64(m.OriginatingAttorneyID),
		ResponsibleAttorneyId:     int64(m.ResponsibleAttorneyID),
		IsDeleted:                 m.IsDeleted,
		MatterNumber:              m.MatterNumber,
		Budget:                    m.Budget,
		HasBudget:                 m.HasBudget,
		Field1:                    m.Field1,
		Field2:                    m.Field2,
		Field3:                    m.Field3,
		DisplayName:               m.DisplayName,
		CreatedById:               int64(m.CreatedByID),
		CreatedOn:                 m.CreatedOn.Format(time.RFC3339),
		ModifiedById:              int64(m.ModifiedByID),
		ModifiedOn:                m.ModifiedOn.Format(time.RFC3339),
		CustomFields:              m.CustomFields,
		CustomFormVersion:         int64(m.CustomFormVersion),
		RetainerFeeBillId:         int64(m.RetainerFeeBillID),
		RetainerFeeFirstPayment:   m.RetainerFeeFirstPayment.Format(time.RFC3339),
		RetainerFeeInitialAmount:  m.RetainerFeeInitialAmount,
		RetainerFeeLastBilledDate: m.RetainerFeeLastBilledDate.Format(time.RFC3339),
		RetainerFeeMonthlyAmount:  m.RetainerFeeMonthlyAmount,
		RetainerFeeUserId:         int64(m.RetainerFeeUserID),
		FirmOfficeId:              int64(m.FirmOfficeID),
		SubjectAreaId:             int64(m.SubjectAreaID),
		IsHidden:                  m.IsHidden,
		LawClerkId:                int64(m.LawClerkID),
	}
}

func fromProto(req *pb.CreateMatterRequest) models.Matter {
	return models.Matter{
		Description:               req.Description,
		OpenDate:                  parseTime(req.OpenDate),
		CloseDate:                 parseTime(req.CloseDate),
		PendingDate:               parseTime(req.PendingDate),
		LimitationDate:            parseTime(req.LimitationDate),
		IsBillable:                req.IsBillable,
		IsLimitationDateSatisfied: req.IsLimitationDateSatisfied,
		StatusID:                  uint(req.StatusId),
		Rate:                      req.Rate,
		PracticeAreaID:            uint(req.PracticeAreaId),
		ClientID:                  uint(req.ClientId),
		OriginatingAttorneyID:     uint(req.OriginatingAttorneyId),
		ResponsibleAttorneyID:     uint(req.ResponsibleAttorneyId),
		IsDeleted:                 req.IsDeleted,
		MatterNumber:              req.MatterNumber,
		Budget:                    req.Budget,
		HasBudget:                 req.HasBudget,
		Field1:                    req.Field1,
		Field2:                    req.Field2,
		Field3:                    req.Field3,
		DisplayName:               req.DisplayName,
		CreatedByID:               uint(req.CreatedById),
		CreatedOn:                 parseTime(req.CreatedOn),
		ModifiedByID:              uint(req.ModifiedById),
		ModifiedOn:                parseTime(req.ModifiedOn),
		CustomFields:              req.CustomFields,
		CustomFormVersion:         uint(req.CustomFormVersion),
		RetainerFeeBillID:         uint(req.RetainerFeeBillId),
		RetainerFeeFirstPayment:   parseTime(req.RetainerFeeFirstPayment),
		RetainerFeeInitialAmount:  req.RetainerFeeInitialAmount,
		RetainerFeeLastBilledDate: parseTime(req.RetainerFeeLastBilledDate),
		RetainerFeeMonthlyAmount:  req.RetainerFeeMonthlyAmount,
		RetainerFeeUserID:         uint(req.RetainerFeeUserId),
		FirmOfficeID:              uint(req.FirmOfficeId),
		SubjectAreaID:             uint(req.SubjectAreaId),
		IsHidden:                  req.IsHidden,
		LawClerkID:                uint(req.LawClerkId),
	}
}

func toProtoActivity(a models.MatterActivity) *pb.MatterActivity {
	return &pb.MatterActivity{
		Id:                int64(a.ID),
		UserId:            int64(a.UserID),
		MatterId:          int64(a.MatterID),
		Date:              a.Date.Format(time.RFC3339),
		Description:       a.Description,
		Rate:              a.Rate,
		CreatedById:       int64(a.CreatedByID),
		CreatedOn:         a.CreatedOn.Format(time.RFC3339),
		ModifiedById:      int64(a.ModifiedByID),
		ModifiedOn:        a.ModifiedOn.Format(time.RFC3339),
		EventEntryId:      int64(a.EventEntryID),
		MatterNoteId:      int64(a.MatterNoteID),
		TaskId:            int64(a.TaskID),
		CategoryId:        int64(a.CategoryID),
		ActivityType:      a.ActivityType,
		Amount:            a.Amount,
		Code:              a.Code,
		MatterId1:         int64(a.MatterID1),
		BillId:            int64(a.BillID),
		Duration:          int64(a.Duration),
		StartedAt:         a.StartedAt.Format(time.RFC3339),
		MatterFlatFeeCode: a.MatterFlatFeeCode,
		IsMain:            a.IsMain,
		Field1:            a.Field1,
		Field2:            a.Field2,
		Field3:            a.Field3,
		IsBillable:        a.IsBillable,
		Charge:            a.Charge,
		NoMatter:          a.NoMatter,
	}
}

func fromProtoActivity(req *pb.CreateMatterActivityRequest) models.MatterActivity {
	return models.MatterActivity{
		UserID:            uint(req.UserId),
		MatterID:          uint(req.MatterId),
		Date:              parseTime(req.Date),
		Description:       req.Description,
		Rate:              req.Rate,
		CreatedByID:       uint(req.CreatedById),
		CreatedOn:         parseTime(req.CreatedOn),
		ModifiedByID:      uint(req.ModifiedById),
		ModifiedOn:        parseTime(req.ModifiedOn),
		EventEntryID:      uint(req.EventEntryId),
		MatterNoteID:      uint(req.MatterNoteId),
		TaskID:            uint(req.TaskId),
		CategoryID:        uint(req.CategoryId),
		ActivityType:      req.ActivityType,
		Amount:            req.Amount,
		Code:              req.Code,
		MatterID1:         uint(req.MatterId1),
		BillID:            uint(req.BillId),
		Duration:          uint(req.Duration),
		StartedAt:         parseTime(req.StartedAt),
		MatterFlatFeeCode: req.MatterFlatFeeCode,
		IsMain:            req.IsMain,
		Field1:            req.Field1,
		Field2:            req.Field2,
		Field3:            req.Field3,
		IsBillable:        req.IsBillable,
		Charge:            req.Charge,
		NoMatter:          req.NoMatter,
	}
}

func toProtoRelated(r models.MatterRelated) *pb.MatterRelated {
	return &pb.MatterRelated{
		Id:            int64(r.ID),
		MatterId:      int64(r.MatterID),
		ActivityLogId: int64(r.ActivityLogID),
	}
}

func fromProtoRelated(req *pb.CreateMatterRelatedRequest) models.MatterRelated {
	return models.MatterRelated{
		MatterID:      uint(req.MatterId),
		ActivityLogID: uint(req.ActivityLogId),
	}
}

func toProtoStatus(s models.MatterStatus) *pb.MatterStatus {
	return &pb.MatterStatus{
		Id:             int64(s.ID),
		Name:           s.Name,
		IsSystem:       s.IsSystem,
		IsNoteRequired: s.IsNoteRequired,
		Color:          s.Color,
	}
}

func fromProtoStatus(req *pb.CreateMatterStatusRequest) models.MatterStatus {
	return models.MatterStatus{
		Name:           req.Name,
		IsSystem:       req.IsSystem,
		IsNoteRequired: req.IsNoteRequired,
		Color:          req.Color,
	}
}

func toProtoCategory(c models.MatterActivityCategory) *pb.MatterActivityCategory {
	return &pb.MatterActivityCategory{
		Id:                        int64(c.ID),
		Name:                      c.Name,
		CreatedById:               int64(c.CreatedByID),
		CreatedOn:                 c.CreatedOn.Format(time.RFC3339),
		Discriminator:             c.Discriminator,
		ModifiedById:              int64(c.ModifiedByID),
		ModifiedOn:                c.ModifiedOn.Format(time.RFC3339),
		Rate:                      c.Rate,
		BillingMethod:             c.BillingMethod,
		CustomRate:                c.CustomRate,
		MatterFlatFeeCategoryRate: c.MatterFlatFeeCategoryRate,
		Field1:                    c.Field1,
		Field2:                    c.Field2,
		Field3:                    c.Field3,
	}
}

func fromProtoCategory(req *pb.CreateMatterActivityCategoryRequest) models.MatterActivityCategory {
	return models.MatterActivityCategory{
		Name:                      req.Name,
		CreatedByID:               uint(req.CreatedById),
		CreatedOn:                 parseTime(req.CreatedOn),
		Discriminator:             req.Discriminator,
		ModifiedByID:              uint(req.ModifiedById),
		ModifiedOn:                parseTime(req.ModifiedOn),
		Rate:                      req.Rate,
		BillingMethod:             req.BillingMethod,
		CustomRate:                req.CustomRate,
		MatterFlatFeeCategoryRate: req.MatterFlatFeeCategoryRate,
		Field1:                    req.Field1,
		Field2:                    req.Field2,
		Field3:                    req.Field3,
	}
}

/*func NewGRPCServer() *GRPCServer {
	return &GRPCServer{
		svc:         service.NewMatterService(),
		activitySvc: service.NewMatterActivityService(),
		relatedSvc:  service.NewMatterRelatedService(),
		statusSvc:   service.NewMatterStatusService(),
		categorySvc: service.NewMatterActivityCategoryService(),
	}
}*/

func (s *GRPCServer) ListMatters(ctx context.Context, req *pb.ListMattersRequest) (*pb.ListMattersResponse, error) {
	matters, err := s.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListMattersResponse{}
	for _, m := range matters {
		resp.Matters = append(resp.Matters, toProto(m))
	}
	return resp, nil
}

func (s *GRPCServer) GetMatter(ctx context.Context, req *pb.GetMatterRequest) (*pb.Matter, error) {
	m, err := s.svc.Get(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return toProto(m), nil
}

func (s *GRPCServer) CreateMatter(ctx context.Context, req *pb.CreateMatterRequest) (*pb.Matter, error) {
	m, err := s.svc.Create(ctx, fromProto(req))
	if err != nil {
		return nil, err
	}
	return toProto(m), nil
}

func (s *GRPCServer) ListMatterActivities(ctx context.Context, req *pb.ListMatterActivitiesRequest) (*pb.ListMatterActivitiesResponse, error) {
	acts, err := s.activitySvc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListMatterActivitiesResponse{}
	for _, a := range acts {
		resp.MatterActivities = append(resp.MatterActivities, toProtoActivity(a))
	}
	return resp, nil
}

func (s *GRPCServer) GetMatterActivity(ctx context.Context, req *pb.GetMatterActivityRequest) (*pb.MatterActivity, error) {
	a, err := s.activitySvc.Get(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return toProtoActivity(a), nil
}

func (s *GRPCServer) CreateMatterActivity(ctx context.Context, req *pb.CreateMatterActivityRequest) (*pb.MatterActivity, error) {
	a, err := s.activitySvc.Create(ctx, fromProtoActivity(req))
	if err != nil {
		return nil, err
	}
	return toProtoActivity(a), nil
}

func (s *GRPCServer) ListMatterRelated(ctx context.Context, req *pb.ListMatterRelatedRequest) (*pb.ListMatterRelatedResponse, error) {
	rels, err := s.relatedSvc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListMatterRelatedResponse{}
	for _, r := range rels {
		resp.MatterRelateds = append(resp.MatterRelateds, toProtoRelated(r))
	}
	return resp, nil
}

func (s *GRPCServer) GetMatterRelated(ctx context.Context, req *pb.GetMatterRelatedRequest) (*pb.MatterRelated, error) {
	r, err := s.relatedSvc.Get(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return toProtoRelated(r), nil
}

func (s *GRPCServer) CreateMatterRelated(ctx context.Context, req *pb.CreateMatterRelatedRequest) (*pb.MatterRelated, error) {
	r, err := s.relatedSvc.Create(ctx, fromProtoRelated(req))
	if err != nil {
		return nil, err
	}
	return toProtoRelated(r), nil
}

func (s *GRPCServer) ListMatterStatuses(ctx context.Context, req *pb.ListMatterStatusesRequest) (*pb.ListMatterStatusesResponse, error) {
	statuses, err := s.statusSvc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListMatterStatusesResponse{}
	for _, st := range statuses {
		resp.MatterStatuses = append(resp.MatterStatuses, toProtoStatus(st))
	}
	return resp, nil
}

func (s *GRPCServer) GetMatterStatus(ctx context.Context, req *pb.GetMatterStatusRequest) (*pb.MatterStatus, error) {
	st, err := s.statusSvc.Get(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return toProtoStatus(st), nil
}

func (s *GRPCServer) CreateMatterStatus(ctx context.Context, req *pb.CreateMatterStatusRequest) (*pb.MatterStatus, error) {
	st, err := s.statusSvc.Create(ctx, fromProtoStatus(req))
	if err != nil {
		return nil, err
	}
	return toProtoStatus(st), nil
}

func (s *GRPCServer) ListMatterActivityCategories(ctx context.Context, req *pb.ListMatterActivityCategoriesRequest) (*pb.ListMatterActivityCategoriesResponse, error) {
	cats, err := s.categorySvc.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListMatterActivityCategoriesResponse{}
	for _, c := range cats {
		resp.MatterActivityCategories = append(resp.MatterActivityCategories, toProtoCategory(c))
	}
	return resp, nil
}

func (s *GRPCServer) GetMatterActivityCategory(ctx context.Context, req *pb.GetMatterActivityCategoryRequest) (*pb.MatterActivityCategory, error) {
	c, err := s.categorySvc.Get(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return toProtoCategory(c), nil
}

func (s *GRPCServer) CreateMatterActivityCategory(ctx context.Context, req *pb.CreateMatterActivityCategoryRequest) (*pb.MatterActivityCategory, error) {
	c, err := s.categorySvc.Create(ctx, fromProtoCategory(req))
	if err != nil {
		return nil, err
	}
	return toProtoCategory(c), nil
}
