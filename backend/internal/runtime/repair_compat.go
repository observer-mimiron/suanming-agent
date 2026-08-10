// Package runtime keeps compatibility aliases for the shared repair contract.
//
// Repair classification, policy and budget ownership lives in internal/repair;
// these aliases let the existing runtime/domain code migrate without changing
// the public behavior or duplicating the contract.
package runtime

import "github.com/observer-mimiron/suanming-agent/internal/repair"

type RepairClass = repair.Class
type RepairAction = repair.Action
type RepairFailure = repair.Failure
type RepairAttempt = repair.Attempt
type RepairState = repair.State
type RepairPolicy = repair.Policy
type RepairDecision = repair.Decision

const (
	RepairTransportTransient = repair.TransportTransient
	RepairTransportFatal     = repair.TransportFatal
	RepairParseError         = repair.ParseError
	RepairSchemaError        = repair.SchemaError
	RepairProjectionMismatch = repair.ProjectionMismatch
	RepairEvidenceOverclaim  = repair.EvidenceOverclaim
	RepairDomainUnauthorized = repair.DomainUnauthorized
	RepairFactConflict       = repair.FactConflict
	RepairMethodContract     = repair.MethodContract
	RepairGuardrailBlocked   = repair.GuardrailBlocked
	RepairUnknown            = repair.Unknown

	RepairActionAccept     = repair.ActionAccept
	RepairActionRetry      = repair.ActionRetry
	RepairActionRepairNode = repair.ActionRepairNode
	RepairActionFallback   = repair.ActionFallback
	RepairActionHardError  = repair.ActionHardError

	DefaultTransportMaxAttempts  = repair.DefaultTransportMaxAttempts
	DefaultJSONParseMaxAttempts  = repair.DefaultJSONParseMaxAttempts
	DefaultMaxTurnRepairAttempts = repair.DefaultMaxTurnRepairAttempts
	DefaultMaxNodeRepairAttempts = repair.DefaultMaxNodeRepairAttempts
)

var DefaultRepairPolicy = repair.DefaultPolicy
var RepairClassForHTTPStatus = repair.ClassForHTTPStatus
var RepairHTTPStatusRetryable = repair.HTTPStatusRetryable
var NewRepairState = repair.NewState
var WithDefaultRepairBudget = repair.WithDefaultBudget
var RepairAttemptsFor = repair.AttemptsFor
var RepairBudgetExhausted = repair.BudgetExhausted
var RecordRepairAttempt = repair.RecordAttempt
