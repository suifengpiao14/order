package order

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// OrderStatus 定义
const (
	Order_status_Pending  = "pending"
	Order_status_Paid     = "paid"
	Order_status_Failed   = "failed"
	Order_status_Canceled = "canceled"
)

// OrderAR 集合根
// 一个抽奖订单，包含一个用户、一个物品
// 根据 freezeID 和 itemID 确认后续步逻辑

type OrderAR struct {
	ID         string    // 唯一标识
	UserID     int64     // 用户 ID
	ItemID     int64     // 商品 ID
	FreezeID   string    // 库存冷冻 ID
	Price      int64     // 单价（分）
	Status     string    // 订单状态
	CreateTime time.Time // 创建时间
	PayTime    *time.Time
}

// OrderRepository 订单仓库
// 保存/查询/修改 订单信息

type OrderRepository interface {
	Save(ctx context.Context, order *OrderAR) error
	FindByID(ctx context.Context, id string) (*OrderAR, error)
	UpdateStatus(ctx context.Context, id string, newStatus string) error
}

// OrderService 订单培育服务
// 创建订单 + 支付确认 + 订单失败操作

type OrderService struct {
	orderRepo OrderRepository
	inv       InventoryGateway // 连接库存确认（无需知道是哪种策略）
}

// InventoryGateway 是 InventoryStrategy 的简化版，用于确认/rollback

type InventoryGateway interface {
	Confirm(ctx context.Context, itemID int64, freezeID string) error
	Release(ctx context.Context, itemID int64, freezeID string) error
}

func NewOrderService(repo OrderRepository, inv InventoryGateway) *OrderService {
	return &OrderService{orderRepo: repo, inv: inv}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID, itemID int64, freezeID string, price int64) (*OrderAR, error) {
	order := &OrderAR{
		ID:         generateOrderID(),
		UserID:     userID,
		ItemID:     itemID,
		FreezeID:   freezeID,
		Price:      price,
		Status:     Order_status_Pending,
		CreateTime: time.Now(),
	}
	if err := s.orderRepo.Save(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *OrderService) ConfirmPay(ctx context.Context, orderID string) error {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != Order_status_Pending {
		return nil
	}
	if err := s.inv.Confirm(ctx, order.ItemID, order.FreezeID); err != nil {
		return err
	}
	now := time.Now()
	order.Status = Order_status_Paid
	order.PayTime = &now
	return s.orderRepo.Save(ctx, order)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != Order_status_Pending {
		return nil
	}
	if err := s.inv.Release(ctx, order.ItemID, order.FreezeID); err != nil {
		return err
	}
	order.Status = Order_status_Canceled
	return s.orderRepo.Save(ctx, order)
}

func generateOrderID() string {
	return uuid.New().String()
}
