# Built-in Payment System Guide

Gabugu (built on Sub2API v0.2.23) includes a built-in payment and checkout system supporting both **balance top-ups** and **subscription plan purchases / renewals**, without deploying a separate payment service.

---

## Table of Contents

- [Supported Payment Providers](#supported-payment-providers)
- [Quick Start](#quick-start)
- [System Settings & Pricing Rules](#system-settings--pricing-rules)
- [Recharge Bonus & Discount Tiers](#recharge-bonus--discount-tiers)
- [Subscription Plans](#subscription-plans)
- [Provider Configuration](#provider-configuration)
- [Provider Instance Management](#provider-instance-management)
- [Webhook Configuration](#webhook-configuration)
- [Payment Flow, Order Statuses & Refunds](#payment-flow-order-statuses--refunds)
- [Migrating from Sub2ApiPay](#migrating-from-sub2apipay)

---

## Supported Payment Providers

| Provider | Payment Methods | Description |
| --- | --- | --- |
| **EasyPay** | Alipay, WeChat Pay | Third-party aggregation via the EasyPay protocol |
| **Alipay (Direct)** | Desktop QR (Face-to-Face preferred, Website Pay fallback), mobile redirect | Direct integration with Alipay Open Platform |
| **WeChat Pay (Direct)** | Native QR, H5, in-WeChat MP/JSAPI | Direct integration with WeChat Pay APIv3 with environment-aware routing |
| **Stripe** | Card, Alipay, WeChat Pay, Link, etc. | International checkout with multi-currency support |
| **Airwallex** | International cards / multi-currency checkout | Embedded Airwallex payment component with multi-currency settlement |

> **Unified frontend methods**: Both direct and EasyPay instances can exist in the backend, but the user checkout always displays unified `Alipay` and `WeChat Pay` buttons. Admins choose one active backing source per visible method so internal provider details stay hidden from end users.

---

## Quick Start

1. Open **Admin → Settings → Payment Settings** (`/admin/settings`)
2. Enable **Payment** and configure the **Site Billing Mode** under **Settings → Features** (`Recharge & Subscription`, `Recharge Only`, or `Subscription Only`)
3. Set amount limits, order timeout, max pending orders, and optional fee rate (leave multipliers at default `1.0` / `0` for Gabugu's 1:1 RMB/USD parity)
4. Add and enable at least one provider instance under **Provider Management**
5. For subscriptions, create and publish plans under **Admin → Orders → Subscription Plans** (`/admin/orders/plans`)
6. Verify `/purchase` as a regular user to confirm currency prefixes, amount preview, and checkout submission

---

## System Settings & Pricing Rules

Configure these options in **Admin → Settings → Payment Settings**:

### Core Limits & Routing

| Setting | Description | Default |
| --- | --- | --- |
| **Enable Payment** | Master payment switch | Off |
| **Product Name Prefix / Suffix** | Order title displayed on external payment pages | - |
| **Min / Max Single Amount** | Per-order amount limits in the payment currency (empty = unlimited) | Min 1 |
| **Daily Limit** | Per-user daily cumulative limit (empty = unlimited) | Unlimited |
| **Order Timeout** | Pending order expiry in minutes (min 1) | 30 |
| **Max Pending Orders** | Maximum concurrent `PENDING` orders per user | 3 |
| **Load Balance Strategy** | `Round Robin` or `Least Amount` (lowest daily volume) | Round Robin |

### Currency, Parity & Fees

- **Internal accounting & Gabugu 1:1 parity**: Account balances, API key quotas, subscription limits, and model prices are denominated in `$` (USD face value). **Gabugu operates at 1:1 RMB/USD parity (`¥1 = $1` account credit, no FX conversion)**, keeping `balance_recharge_multiplier = 1.0` and `subscription_usd_to_cny_rate = 0`.
- **Checkout payment currency**: Defaults to `CNY` (`¥`); Stripe and Airwallex instances can configure other ISO currencies (`USD`, `EUR`, `HKD`, etc.). The amount input prefix and submit button automatically reflect the active payment currency symbol.
- **Balance recharge multiplier (`balance_recharge_multiplier`, optional)**:
  - Formula: `creditedBase = round2(paymentAmount × multiplier)` (up to 6 decimals; **default `1.0`** — keep at `1.0` for 1:1 parity).
  - Only when set to a value other than `1` does checkout display the conversion preview line.
- **Processing fee (`recharge_fee_rate`)**:
  - Added on top of the payment base and rounded up to the currency's smallest unit: `payAmount = payBase + ceil(payBase × feeRate / 100)`.
- **Subscription CNY multiplier (`subscription_usd_to_cny_rate`, optional)**:
  - **Default `0`** (disabled; charges the plan's listed price 1:1 — keep at `0` on Gabugu). When `> 0` and the active channel currency is `CNY`, computes `round(planPrice × rate)` before fee.

### Cancel Rate Limiting & Help Content

- **Cancel Rate Limiting**: Sliding or fixed window cap to prevent repeated order creation and cancellation abuse.
- **Help Image & Markdown Text**: Displayed below the checkout form for currency notes, support contact, and troubleshooting guidance.

---

## Recharge Bonus & Discount Tiers

Configure tiered recharge promotions (matching the highest qualifying `min_amount` tier) plus an optional Markdown announcement banner:

- **Bonus mode (`bonus`)**: User pays the full entered amount and receives extra USD credit (`bonus = round2(baseUSD × percent%)`; `credited = baseUSD + bonus`).
- **Discount mode (`discount`)**: User pays a discounted gateway amount while receiving the full base USD credit.
- **Affiliate rebate basis**: Affiliate rebates for balance recharges are calculated strictly on paid principal (`amount - bonus_amount`), excluding bonus credits.

---

## Subscription Plans

Manage purchasable plans under **Admin → Orders → Subscription Plans** (`/admin/orders/plans`):

- Each plan binds to a **subscription-type group**, inheriting its platform, allowed models, rate multiplier, optional peak-hour multiplier, and daily/weekly/monthly USD quota limits.
- Supports name, original/sale price, validity period, feature bullets, display order, and `for_sale` visibility.
- Active renewals extend `expires_at` without resetting current window usage; renewals after expiry start a fresh quota cycle.

---

## Provider Configuration

Add credentials in **Payment Settings → Provider Management → Add Provider**. Notify and Return URLs are auto-constructed from your configured domain.

### EasyPay
- Required: **Merchant ID (PID)**, **Merchant Key (PKey)**, **API Base URL**
- Optional: **Alipay Channel ID**, **WeChat Channel ID**

### Alipay (Direct)
- Required: **AppID**, **RSA2 Private Key**, **Alipay Public Key**
- Desktop prefers Face-to-Face precreate QR with automatic Computer Website Pay fallback; mobile returns a WAP/app redirect URL.

### WeChat Pay (Direct)
- Required: **AppID**, **Merchant ID (MchID)**, **Merchant API Private Key (PEM)**, **32-byte APIv3 Key**, **WeChat Pay Public Key (PEM)**, **Public Key ID**, **Certificate Serial Number**
- Automatically routes desktop Native QR, mobile browser H5, and in-WeChat MP/JSAPI.

### Stripe
- Required: **Secret Key** (`sk_live_...` / `sk_test_...`), **Publishable Key** (`pk_live_...` / `pk_test_...`), **Webhook Secret** (`whsec_...`)
- Subscribe Stripe webhooks to `payment_intent.succeeded` and `payment_intent.payment_failed`.

### Airwallex
- Required: **Client ID**, **API Key**, **Webhook Secret**, currency & environment settings.

---

## Provider Instance Management

Create multiple instances per provider type for failover and risk limits:
- Independent **min/max single amount** and **daily volume limit** (instances over limit are automatically skipped)
- Independent enable/disable toggle and drag-to-reorder priority
- Per-instance **self-service refund eligibility** (`allow_user_refund`)

---

## Webhook Configuration

Use **HTTPS** and allow inbound callbacks from your payment gateways:

| Provider | Callback Endpoint |
| --- | --- |
| **EasyPay** | `https://your-domain.com/api/v1/payment/webhook/easypay` |
| **Alipay (Direct)** | `https://your-domain.com/api/v1/payment/webhook/alipay` |
| **WeChat Pay (Direct)** | `https://your-domain.com/api/v1/payment/webhook/wxpay` |
| **Stripe** | `https://your-domain.com/api/v1/payment/webhook/stripe` |
| **Airwallex** | `https://your-domain.com/api/v1/payment/webhook/airwallex` |

---

## Payment Flow, Order Statuses & Refunds

### Flow
1. User selects an amount or plan and payment method → backend validates limits, conversion rate, and creates a `PENDING` order on a load-balanced provider instance.
2. User pays → webhook signature and amount (tolerance `0.01` or half smallest unit) are verified → order moves to `PAID` (late callbacks within a 5-minute grace window after expiry or cancellation are reconciled).
3. Fulfillment leases the order (`RECHARGING`, 5-minute lease), atomically credits USD balance or activates/extends the subscription → `COMPLETED`.
4. A background job runs every 60 seconds to query upstream payment status before expiring timed-out `PENDING` orders.

### Order Status Reference

| Status | Description |
| --- | --- |
| `PENDING` | Awaiting user payment |
| `PAID` | Payment confirmed, awaiting fulfillment |
| `RECHARGING` | Fulfillment in progress |
| `COMPLETED` | Balance credited or subscription activated |
| `EXPIRED` | Timed out without payment |
| `CANCELLED` | Cancelled by user |
| `FAILED` | Fulfillment failed (admin can retry) |
| `REFUND_REQUESTED` | Refund requested by user, awaiting review |
| `REFUNDING` / `REFUND_PENDING` | Refund processing / awaiting provider confirmation |
| `REFUNDED` / `PARTIALLY_REFUNDED` | Refund completed (full / partial) |
| `REFUND_FAILED` | Refund failed |

### Refund Rules
- **User self-service refunds (`RequestRefund`)**: Supported **only for balance recharge orders** (subscription orders return `INVALID_ORDER_TYPE`). Self-service requests are full-order only and require the user's current USD balance to be at least the order's total credited USD (`order.amount`, including any bonus credits).
- **Admin refunds (`PrepareRefund` / `ExecuteRefund`)**:
  - **Balance orders**: Supports full or partial refunds specified in **credited USD** (`<=` remaining `order.amount`). Admins can toggle `deduct_balance` or use `force=true` when the user's current balance is lower than the refund amount.
  - **Subscription orders**: Supported by deducting subscription days (`subscription_days`) or revoking the subscription.
  - **Gateway payout proration**: Actual payout in the original payment currency is prorated from `pay_amount` by `refund_usd / order.amount` (capped at `pay_amount` on full refund), ensuring promotional bonus credits never pay out extra cash.

---

## Migrating from Sub2ApiPay

| Aspect | Legacy Sub2ApiPay | Built-in Payment (v0.2.23) |
| --- | --- | --- |
| Deployment | Separate Next.js + PostgreSQL service | Built directly into the backend |
| Providers | EasyPay, Alipay, WeChat, Stripe | EasyPay, Alipay Direct, WeChat Direct, Stripe, **Airwallex** |
| Subscription Plans | Supported | **Fully supported** (purchase, renewal, USD→CNY rate) |
| Promotions | External config | **Built-in** bonus/discount tiers & conversion rates |
| Fulfillment | External Admin API callback | Internal atomic transaction + auto-reconciliation |
| Order Management | Separate admin UI | Integrated under **Admin → Orders** (`/admin/orders`) |

Migration steps: enter credentials in **Settings → Payment Settings** → point gateway webhooks to the built-in paths → verify one small order end-to-end → keep legacy Sub2ApiPay read-only for historical lookup before decommissioning.
