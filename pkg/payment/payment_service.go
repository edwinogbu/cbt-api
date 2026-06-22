package payment

import (
    "context"
    "fmt"
    "os"

    "cbt-api/internal/models"
)

type PaymentService struct {
    gateways map[models.PaymentGateway]Gateway
}

func NewPaymentService() *PaymentService {
    svc := &PaymentService{
        gateways: make(map[models.PaymentGateway]Gateway),
    }
    
    // Initialize gateways only if credentials are present
    if os.Getenv("STRIPE_SECRET_KEY") != "" {
        svc.gateways[models.GatewayStripe] = NewStripeGateway()
    }
    if os.Getenv("PAYSTACK_SECRET_KEY") != "" {
        svc.gateways[models.GatewayPaystack] = NewPaystackGateway()
    }
    if os.Getenv("FLUTTERWAVE_SECRET_KEY") != "" {
        svc.gateways[models.GatewayFlutterwave] = NewFlutterwaveGateway()
    }
    
    // At least one gateway should be available
    if len(svc.gateways) == 0 {
        // For development, initialize Stripe with test mode
        svc.gateways[models.GatewayStripe] = NewStripeGateway()
    }
    
    return svc
}

func (s *PaymentService) CreatePayment(ctx context.Context, gateway models.PaymentGateway, req *PaymentRequest) (*models.PaymentIntent, error) {
    gatewayImpl, ok := s.gateways[gateway]
    if !ok {
        return nil, fmt.Errorf("unsupported gateway: %s", gateway)
    }
    return gatewayImpl.CreatePayment(ctx, req)
}

func (s *PaymentService) VerifyPayment(ctx context.Context, gateway models.PaymentGateway, reference string) (*PaymentVerification, error) {
    gatewayImpl, ok := s.gateways[gateway]
    if !ok {
        return nil, fmt.Errorf("unsupported gateway: %s", gateway)
    }
    return gatewayImpl.VerifyPayment(ctx, reference)
}

func (s *PaymentService) CreateSubscription(ctx context.Context, gateway models.PaymentGateway, req *SubscriptionRequest) (*SubscriptionResult, error) {
    gatewayImpl, ok := s.gateways[gateway]
    if !ok {
        return nil, fmt.Errorf("unsupported gateway: %s", gateway)
    }
    return gatewayImpl.CreateSubscription(ctx, req)
}

func (s *PaymentService) CancelSubscription(ctx context.Context, gateway models.PaymentGateway, subscriptionID string) error {
    gatewayImpl, ok := s.gateways[gateway]
    if !ok {
        return fmt.Errorf("unsupported gateway: %s", gateway)
    }
    return gatewayImpl.CancelSubscription(ctx, subscriptionID)
}

func (s *PaymentService) GetSubscription(ctx context.Context, gateway models.PaymentGateway, subscriptionID string) (*SubscriptionStatus, error) {
    gatewayImpl, ok := s.gateways[gateway]
    if !ok {
        return nil, fmt.Errorf("unsupported gateway: %s", gateway)
    }
    return gatewayImpl.GetSubscription(ctx, subscriptionID)
}

func (s *PaymentService) GetAvailableGateways() []models.PaymentGateway {
    gateways := []models.PaymentGateway{}
    for g := range s.gateways {
        gateways = append(gateways, g)
    }
    return gateways
}

func (s *PaymentService) GetGateway(gateway models.PaymentGateway) (Gateway, error) {
    g, ok := s.gateways[gateway]
    if !ok {
        return nil, fmt.Errorf("unsupported gateway: %s", gateway)
    }
    return g, nil
}


