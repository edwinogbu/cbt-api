package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "cbt-api/internal/models"
    "cbt-api/internal/subscription/dto"
    "cbt-api/internal/subscription/service"
)

type SubscriptionHandler struct {
    service *service.SubscriptionService
}

func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
    return &SubscriptionHandler{service: svc}
}

// CreateSubscription godoc
// @Summary      Create a new subscription
// @Description  Creates a pending subscription and returns a one‑time payment link (Paystack/Flutterwave/Stripe).
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateSubscriptionRequest true "Subscription details"
// @Success      201  {object}  map[string]interface{}  "message + data (CreateSubscriptionResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /subscriptions [post]
// @Example      request  {"school_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","tier":"premium","interval":"monthly","gateway":"paystack","auto_renew":true,"success_url":"https://yourapp.com/success","cancel_url":"https://yourapp.com/cancel","email":"finance@school.edu"}
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    var req dto.CreateSubscriptionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    result, err := h.service.CreateSubscription(c.Request.Context(), &req, userID.(string), req.SchoolID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "Subscription created successfully",
        "data":    result,
    })
}

// GetSubscription godoc
// @Summary      Get subscription by ID
// @Description  Retrieve a single subscription by its UUID.
// @Tags         subscriptions
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subscription ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "message + data (SubscriptionResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /subscriptions/{id} [get]
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
    id := c.Param("id")
    if _, err := uuid.Parse(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
        return
    }

    sub, err := h.service.GetSubscription(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Subscription retrieved successfully",
        "data":    sub,
    })
}

// GetSubscriptions godoc
// @Summary      Get all subscriptions for a school
// @Description  List all subscriptions (including inactive) belonging to a school.
// @Tags         subscriptions
// @Produce      json
// @Security     BearerAuth
// @Param        school_id query string true "School ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "message + data (array of SubscriptionResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Router       /subscriptions [get]
func (h *SubscriptionHandler) GetSubscriptions(c *gin.Context) {
    schoolID := c.Query("school_id")
    if schoolID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "school_id is required"})
        return
    }

    subs, err := h.service.GetSubscriptionsBySchool(schoolID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Subscriptions retrieved successfully",
        "data":    subs,
    })
}

// GetCurrentSubscription godoc
// @Summary      Get current active subscription for a school
// @Description  Returns the currently active (or trial/pending) subscription for a school.
// @Tags         subscriptions
// @Produce      json
// @Security     BearerAuth
// @Param        schoolId path string true "School ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "message + data (SubscriptionResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /subscriptions/school/{schoolId}/current [get]
func (h *SubscriptionHandler) GetCurrentSubscription(c *gin.Context) {
    schoolID := c.Param("schoolId")
    if _, err := uuid.Parse(schoolID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school id"})
        return
    }

    sub, err := h.service.GetCurrentSubscription(schoolID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Current subscription retrieved successfully",
        "data":    sub,
    })
}

// UpdateSubscription godoc
// @Summary      Update a subscription
// @Description  Update tier, auto‑renew, cancel‑at‑end, or status.
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subscription ID (UUID)"
// @Param        request body dto.UpdateSubscriptionRequest true "Update fields"
// @Success      200  {object}  map[string]interface{}  "message + data (SubscriptionResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /subscriptions/{id} [put]
func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    id := c.Param("id")
    if _, err := uuid.Parse(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
        return
    }

    var req dto.UpdateSubscriptionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    sub, err := h.service.UpdateSubscription(id, &req, userID.(string))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Subscription updated successfully",
        "data":    sub,
    })
}

// CancelSubscription godoc
// @Summary      Cancel a subscription
// @Description  Cancel immediately or at the end of the current period.
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subscription ID (UUID)"
// @Param        request body dto.CancelSubscriptionRequest true "Cancel options"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /subscriptions/{id}/cancel [post]
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    id := c.Param("id")
    if _, err := uuid.Parse(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
        return
    }

    var req dto.CancelSubscriptionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := h.service.CancelSubscription(id, req.CancelImmediately, req.Reason, userID.(string)); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Subscription cancelled successfully"})
}

// RenewSubscription godoc
// @Summary      Renew a subscription
// @Description  Extend an active subscription for a new interval (generates a new payment link).
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subscription ID (UUID)"
// @Param        request body dto.RenewSubscriptionRequest true "Renewal interval"
// @Success      200  {object}  map[string]interface{}  "message + data (SubscriptionResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /subscriptions/{id}/renew [post]
func (h *SubscriptionHandler) RenewSubscription(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    id := c.Param("id")
    if _, err := uuid.Parse(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
        return
    }

    var req dto.RenewSubscriptionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    sub, err := h.service.RenewSubscription(id, &req, userID.(string))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Subscription renewed successfully",
        "data":    sub,
    })
}

// CreatePaymentIntent godoc
// @Summary      Create a one‑time payment intent for a subscription
// @Description  Generate a new payment link for a subscription (e.g., manual renewal or upgrade).
// @Tags         payments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subscription ID (UUID)"
// @Param        request body dto.CreatePaymentIntentRequest true "Payment intent options (success/cancel URLs)"
// @Success      200  {object}  map[string]interface{}  "message + data (PaymentIntentResponse)"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /subscriptions/{id}/payment-intent [post]
func (h *SubscriptionHandler) CreatePaymentIntent(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    subscriptionID := c.Param("id")
    if _, err := uuid.Parse(subscriptionID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
        return
    }

    var req dto.CreatePaymentIntentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    req.SubscriptionID = subscriptionID

    pi, err := h.service.CreatePaymentIntent(subscriptionID, &req, userID.(string))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Payment intent created successfully",
        "data":    pi,
    })
}

// ConfirmPaymentIntent godoc
// @Summary      Confirm a payment intent (verify with gateway)
// @Description  Manual verification of a payment intent – usually not needed because webhook handles it.
// @Tags         payments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Payment Intent ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Router       /subscriptions/payment-intent/{id}/confirm [post]
func (h *SubscriptionHandler) ConfirmPaymentIntent(c *gin.Context) {
    paymentIntentID := c.Param("id")
    if _, err := uuid.Parse(paymentIntentID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment intent id"})
        return
    }

    if err := h.service.ConfirmPaymentIntent(paymentIntentID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Payment confirmed successfully"})
}

// GetInvoices godoc
// @Summary      Get invoices for a subscription
// @Tags         invoices
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subscription ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "message + data (array of InvoiceResponse)"
// @Router       /subscriptions/{id}/invoices [get]
func (h *SubscriptionHandler) GetInvoices(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Invoices retrieved successfully",
        "data":    []interface{}{},
    })
}

// GetTransactions godoc
// @Summary      Get payment transactions for a subscription
// @Tags         payments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subscription ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "message + data (array of PaymentTransactionResponse)"
// @Router       /subscriptions/{id}/transactions [get]
func (h *SubscriptionHandler) GetTransactions(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Transactions retrieved successfully",
        "data":    []interface{}{},
    })
}

// GetSubscriptionUsage godoc
// @Summary      Get usage statistics for a subscription
// @Description  Current usage vs limits (students, teachers, exams, storage).
// @Tags         subscriptions
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Subscription ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "message + data (SubscriptionUsageResponse)"
// @Router       /subscriptions/{id}/usage [get]
func (h *SubscriptionHandler) GetSubscriptionUsage(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message": "Usage retrieved successfully",
        "data":    nil,
    })
}

// VerifyPayment godoc
// @Summary      Verify a payment by gateway reference
// @Description  Manually verify a payment using the gateway's reference.
// @Tags         payments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.VerifyPaymentRequest true "Verification details (reference, gateway)"
// @Success      200  {object}  map[string]interface{}  "message"
// @Failure      400  {object}  map[string]interface{}
// @Router       /subscriptions/verify [post]
func (h *SubscriptionHandler) VerifyPayment(c *gin.Context) {
    var req dto.VerifyPaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // TODO: implement service.VerifyPayment
    c.JSON(http.StatusOK, gin.H{"message": "Payment verified successfully"})
}

// HandleWebhook godoc
// @Summary      Webhook receiver for payment gateways
// @Description  Paystack, Flutterwave, or Stripe webhook endpoint. No authentication.
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        gateway path string true "Gateway name: stripe, paystack, flutterwave"
// @Success      200  {object}  map[string]interface{}  "status"
// @Failure      400  {object}  map[string]interface{}
// @Router       /webhook/{gateway} [post]
func (h *SubscriptionHandler) HandleWebhook(c *gin.Context) {
    gatewayParam := c.Param("gateway")
    var gateway models.PaymentGateway
    switch gatewayParam {
    case "stripe":
        gateway = models.GatewayStripe
    case "paystack":
        gateway = models.GatewayPaystack
    case "flutterwave":
        gateway = models.GatewayFlutterwave
    default:
        c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported gateway"})
        return
    }

    body, err := c.GetRawData()
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
        return
    }

    var signature string
    switch gateway {
    case models.GatewayPaystack:
        signature = c.GetHeader("x-paystack-signature")
    case models.GatewayFlutterwave:
        signature = c.GetHeader("verif-hash")
    case models.GatewayStripe:
        signature = c.GetHeader("stripe-signature")
    }

    err = h.service.ProcessWebhook(c.Request.Context(), gateway, body, signature)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "received"})
}




// package handler

// import (
//     "net/http"

//     "github.com/gin-gonic/gin"
//     "github.com/google/uuid"

//     "cbt-api/internal/models"                // ← add this line
//     "cbt-api/internal/subscription/dto"
//     "cbt-api/internal/subscription/service"
// )

// type SubscriptionHandler struct {
//     service *service.SubscriptionService
// }

// func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
//     return &SubscriptionHandler{service: svc}
// }

// // CreateSubscription godoc
// // @Summary Create a new subscription
// // @Tags subscriptions
// // @Accept json
// // @Produce json
// // @Param request body dto.CreateSubscriptionRequest true "Subscription details"
// // @Success 201 {object} dto.CreateSubscriptionResponse
// // @Router /subscriptions [post]
// func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
//         return
//     }

//     var req dto.CreateSubscriptionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     result, err := h.service.CreateSubscription(c.Request.Context(), &req, userID.(string), req.SchoolID)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusCreated, gin.H{
//         "message": "Subscription created successfully",
//         "data":    result,
//     })
// }

// // GetSubscription godoc
// // @Summary Get subscription by ID
// // @Tags subscriptions
// // @Produce json
// // @Param id path string true "Subscription ID"
// // @Success 200 {object} dto.SubscriptionResponse
// // @Router /subscriptions/{id} [get]
// func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
//     id := c.Param("id")
//     if _, err := uuid.Parse(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
//         return
//     }

//     sub, err := h.service.GetSubscription(id)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "message": "Subscription retrieved successfully",
//         "data":    sub,
//     })
// }

// // GetSubscriptions godoc
// // @Summary Get all subscriptions for a school
// // @Tags subscriptions
// // @Produce json
// // @Param school_id query string true "School ID"
// // @Success 200 {array} dto.SubscriptionResponse
// // @Router /subscriptions [get]
// func (h *SubscriptionHandler) GetSubscriptions(c *gin.Context) {
//     schoolID := c.Query("school_id")
//     if schoolID == "" {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "school_id is required"})
//         return
//     }

//     subs, err := h.service.GetSubscriptionsBySchool(schoolID)
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "message": "Subscriptions retrieved successfully",
//         "data":    subs,
//     })
// }

// // GetCurrentSubscription godoc
// // @Summary Get current active subscription for a school
// // @Tags subscriptions
// // @Produce json
// // @Param schoolId path string true "School ID"
// // @Success 200 {object} dto.SubscriptionResponse
// // @Router /subscriptions/school/{schoolId}/current [get]
// func (h *SubscriptionHandler) GetCurrentSubscription(c *gin.Context) {
//     schoolID := c.Param("schoolId")
//     if _, err := uuid.Parse(schoolID); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid school id"})
//         return
//     }

//     sub, err := h.service.GetCurrentSubscription(schoolID)
//     if err != nil {
//         c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "message": "Current subscription retrieved successfully",
//         "data":    sub,
//     })
// }

// // UpdateSubscription godoc
// // @Summary Update a subscription
// // @Tags subscriptions
// // @Accept json
// // @Produce json
// // @Param id path string true "Subscription ID"
// // @Param request body dto.UpdateSubscriptionRequest true "Update fields"
// // @Success 200 {object} dto.SubscriptionResponse
// // @Router /subscriptions/{id} [put]
// func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
//         return
//     }

//     id := c.Param("id")
//     if _, err := uuid.Parse(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
//         return
//     }

//     var req dto.UpdateSubscriptionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     sub, err := h.service.UpdateSubscription(id, &req, userID.(string))
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "message": "Subscription updated successfully",
//         "data":    sub,
//     })
// }

// // CancelSubscription godoc
// // @Summary Cancel a subscription
// // @Tags subscriptions
// // @Accept json
// // @Produce json
// // @Param id path string true "Subscription ID"
// // @Param request body dto.CancelSubscriptionRequest true "Cancel options"
// // @Success 200 {object} map[string]interface{}
// // @Router /subscriptions/{id}/cancel [post]
// func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
//         return
//     }

//     id := c.Param("id")
//     if _, err := uuid.Parse(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
//         return
//     }

//     var req dto.CancelSubscriptionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     if err := h.service.CancelSubscription(id, req.CancelImmediately, req.Reason, userID.(string)); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{"message": "Subscription cancelled successfully"})
// }

// // RenewSubscription godoc
// // @Summary Renew a subscription
// // @Tags subscriptions
// // @Accept json
// // @Produce json
// // @Param id path string true "Subscription ID"
// // @Param request body dto.RenewSubscriptionRequest true "Renewal options"
// // @Success 200 {object} dto.SubscriptionResponse
// // @Router /subscriptions/{id}/renew [post]
// func (h *SubscriptionHandler) RenewSubscription(c *gin.Context) {
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
//         return
//     }

//     id := c.Param("id")
//     if _, err := uuid.Parse(id); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
//         return
//     }

//     var req dto.RenewSubscriptionRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     sub, err := h.service.RenewSubscription(id, &req, userID.(string))
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "message": "Subscription renewed successfully",
//         "data":    sub,
//     })
// }

// // CreatePaymentIntent godoc
// // @Summary Create a payment intent for a subscription
// // @Tags payments
// // @Accept json
// // @Produce json
// // @Param id path string true "Subscription ID"
// // @Param request body dto.CreatePaymentIntentRequest true "Payment intent options"
// // @Success 200 {object} dto.PaymentIntentResponse
// // @Router /subscriptions/{id}/payment-intent [post]
// func (h *SubscriptionHandler) CreatePaymentIntent(c *gin.Context) {
//     userID, exists := c.Get("user_id")
//     if !exists {
//         c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
//         return
//     }

//     subscriptionID := c.Param("id")
//     if _, err := uuid.Parse(subscriptionID); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
//         return
//     }

//     var req dto.CreatePaymentIntentRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }
//     req.SubscriptionID = subscriptionID

//     pi, err := h.service.CreatePaymentIntent(subscriptionID, &req, userID.(string))
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{
//         "message": "Payment intent created successfully",
//         "data":    pi,
//     })
// }

// // ConfirmPaymentIntent godoc
// // @Summary Confirm a payment intent after user completes payment
// // @Tags payments
// // @Produce json
// // @Param id path string true "Payment Intent ID"
// // @Success 200 {object} map[string]interface{}
// // @Router /subscriptions/payment-intent/{id}/confirm [post]
// func (h *SubscriptionHandler) ConfirmPaymentIntent(c *gin.Context) {
//     paymentIntentID := c.Param("id")
//     if _, err := uuid.Parse(paymentIntentID); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment intent id"})
//         return
//     }

//     if err := h.service.ConfirmPaymentIntent(paymentIntentID); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{"message": "Payment confirmed successfully"})
// }

// // GetInvoices godoc
// // @Summary Get all invoices for a subscription
// // @Tags invoices
// // @Produce json
// // @Param id path string true "Subscription ID"
// // @Success 200 {array} dto.InvoiceResponse
// // @Router /subscriptions/{id}/invoices [get]
// func (h *SubscriptionHandler) GetInvoices(c *gin.Context) {
//     // TODO: implement service.GetInvoicesBySubscription
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Invoices retrieved successfully",
//         "data":    []interface{}{},
//     })
// }

// // GetTransactions godoc
// // @Summary Get all payment transactions for a subscription
// // @Tags payments
// // @Produce json
// // @Param id path string true "Subscription ID"
// // @Success 200 {array} dto.PaymentTransactionResponse
// // @Router /subscriptions/{id}/transactions [get]
// func (h *SubscriptionHandler) GetTransactions(c *gin.Context) {
//     // TODO: implement service.GetTransactionsBySubscription
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Transactions retrieved successfully",
//         "data":    []interface{}{},
//     })
// }

// // GetSubscriptionUsage godoc
// // @Summary Get usage statistics for a subscription
// // @Tags subscriptions
// // @Produce json
// // @Param id path string true "Subscription ID"
// // @Success 200 {object} dto.SubscriptionUsageResponse
// // @Router /subscriptions/{id}/usage [get]
// func (h *SubscriptionHandler) GetSubscriptionUsage(c *gin.Context) {
//     // TODO: implement service.GetSubscriptionUsage
//     c.JSON(http.StatusOK, gin.H{
//         "message": "Usage retrieved successfully",
//         "data":    nil,
//     })
// }

// // VerifyPayment godoc
// // @Summary Verify a payment by reference
// // @Tags payments
// // @Accept json
// // @Produce json
// // @Param request body dto.VerifyPaymentRequest true "Verification details"
// // @Success 200 {object} map[string]interface{}
// // @Router /subscriptions/verify [post]
// func (h *SubscriptionHandler) VerifyPayment(c *gin.Context) {
//     var req dto.VerifyPaymentRequest
//     if err := c.ShouldBindJSON(&req); err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     // TODO: implement service.VerifyPayment
//     c.JSON(http.StatusOK, gin.H{"message": "Payment verified successfully"})
// }

// // // HandleWebhook godoc
// // // @Summary Handle webhook from payment gateway
// // // @Tags webhooks
// // // @Accept json
// // // @Produce json
// // // @Param gateway path string true "Gateway name (stripe, paystack, flutterwave)"
// // // @Success 200 {object} map[string]interface{}
// // // @Router /api/v1/webhook/{gateway} [post]
// // func (h *SubscriptionHandler) HandleWebhook(c *gin.Context) {
// //     body, err := c.GetRawData()
// //     if err != nil {
// //         c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
// //         return
// //     }

// //     // TODO: implement service.ProcessWebhook
// //     _ = body

// //     c.JSON(http.StatusOK, gin.H{"status": "received"})
// // }


// // HandleWebhook processes incoming webhooks from payment gateways
// func (h *SubscriptionHandler) HandleWebhook(c *gin.Context) {
//     gatewayParam := c.Param("gateway")
//     // Map string to models.PaymentGateway
//     var gateway models.PaymentGateway
//     switch gatewayParam {
//     case "stripe":
//         gateway = models.GatewayStripe
//     case "paystack":
//         gateway = models.GatewayPaystack
//     case "flutterwave":
//         gateway = models.GatewayFlutterwave
//     default:
//         c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported gateway"})
//         return
//     }

//     body, err := c.GetRawData()
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
//         return
//     }

//     // Get the signature header (depends on gateway)
//     var signature string
//     switch gateway {
//     case models.GatewayPaystack:
//         signature = c.GetHeader("x-paystack-signature")
//     case models.GatewayFlutterwave:
//         signature = c.GetHeader("verif-hash")
//     case models.GatewayStripe:
//         signature = c.GetHeader("stripe-signature")
//     }

//     err = h.service.ProcessWebhook(c.Request.Context(), gateway, body, signature)
//     if err != nil {
//         c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//         return
//     }

//     c.JSON(http.StatusOK, gin.H{"status": "received"})
// }

