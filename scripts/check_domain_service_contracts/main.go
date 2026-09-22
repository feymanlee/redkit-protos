package main

// 本文件校验跨领域 Proto service 仅声明已批准的 RPC 集合，防止契约边界在生成前无意扩张。

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	packagePattern = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z0-9_.]+)\s*;`)
	servicePattern = regexp.MustCompile(`\bservice\s+([A-Za-z][A-Za-z0-9_]*)\s*\{`)
	rpcPattern     = regexp.MustCompile(`\brpc\s+([A-Za-z][A-Za-z0-9_]*)\s*\(`)
	httpPattern    = regexp.MustCompile(`\b(get|post|put|delete|patch)\s*:\s*"([^"]+)"`)
)

type serviceContract map[string][]string

var businessContracts = map[string]serviceContract{
	"ops.reward.v1": {
		"RewardClosureService": {
			"PrepareClosure", "ApplyClosure", "CancelClosure", "GetClosureStatus",
		},
		"RewardPackageService": {
			"ListRewardPackages", "GetRewardPackage", "CreateRewardPackage", "CreateRewardPackageRevisionDraft",
			"UpdateRewardPackageDraft", "PreflightRewardPackageDraft", "PublishRewardPackage", "DisableRewardPackage",
			"DiscardRewardPackageDraft", "ResolvePublishedRewardPackage",
		},
		"RewardService": {
			"GrantReward", "GetRewardGrant", "ListRewardGrants", "GetRewardGrantObservation", "CreateManualRewardGrant",
			"ListActionRequiredRewardItems", "ReplayRewardItem", "ListPendingManualRewardGrants", "ApproveManualRewardGrant",
			"RejectManualRewardGrant", "WithdrawManualRewardGrant",
		},
	},
	"user.consumer.v1": {
		"ConsumerUserDeletionService": {
			"BeginUserDeletion", "GetUserDeletion", "CancelUserDeletion",
		},
		"ConsumerAuthenticationService": {
			"GetAuthenticationOptions", "Register", "Login", "ResendLoginChallenge", "CompleteLoginChallenge",
			"RefreshToken", "Logout", "StartPasswordReset", "CompletePasswordResetChallenge",
			"CompletePasswordReset", "CompleteRequiredPasswordReset",
		},
		"ConsumerCredentialService": {
			"ListCredentials", "BindPhone", "CompleteRequiredPhoneBinding", "ChangePhone", "UnbindPhone",
			"SetPassword", "ChangePassword", "LinkExternalIdentity", "UnlinkExternalIdentity",
		},
		"ConsumerProfileService": {"GetCurrentUser", "UpdateProfile", "GetPublicProfile", "ChangeUserCode"},
		"ConsumerRecoveryService": {
			"StartUserRecovery", "GetUserRecovery", "CancelUserRecovery",
			"CreateTrustedDeviceRecoveryChallenge", "VerifyTrustedDeviceRecoveryChallenge",
			"SubmitPaymentRecoveryEvidence", "ClaimRecoveryGrant", "CompleteUserRecovery",
		},
		"ConsumerSecurityService": {
			"GetSecurityOverview", "StartStepUpAuthorization", "ResendStepUpChallenge",
			"CompleteStepUpAuthorization", "BeginTotpEnrollment", "ConfirmTotpEnrollment",
			"BeginRequiredTotpEnrollment", "ConfirmRequiredTotpEnrollment", "RegenerateRecoveryCodes",
			"DisableTotp", "ListSecurityActivities",
		},
		"ConsumerSessionService": {
			"ListSessions", "RevokeSession", "RevokeOtherSessions", "RevokeAllSessions", "ListDevices",
			"RevokeDevice", "RegisterCurrentDeviceRecoveryKey", "RemoveCurrentDeviceRecoveryKey",
		},
	},
	"user.internal.v1": {
		"UserIdentityQueryService":        {"ResolveUserIdentity", "BatchResolveUsers", "BatchGetPublicProfiles"},
		"UserSessionIntrospectionService": {"CheckSession", "GetAccessTokenVerificationKeys"},
	},
	"user.administration.v1": {
		"UserAdministrationService": {
			"ChangeUserCode",
			"CreateUser",
			"GetUser",
			"GetUserDeletion",
			"ListUsers",
			"BeginAdministrativeUserDeletion",
			"CancelAdministrativeUserDeletion",
			"RetryAdministrativeUserDeletion",
			"ModerateUserProfile",
			"ReactivateUser",
			"RequirePasswordReset",
			"ResetMFA",
			"RevokeDeviceTrust",
			"ResetUserCode",
			"RevokeCredential",
			"SearchUsers",
			"SuspendUser",
		},
		"UserCredentialService": {
			"ListCredentials",
			"BindCredential",
			"UnbindCredential",
			"ChangePassword",
			"ResetPassword",
			"ListCredentialPII",
		},
		"UserSessionService": {"ListSessions", "CheckSession", "RevokeSession", "RevokeAllSessions"},
		"UserDeviceService":  {"ListDevices", "TrustDevice", "RegisterRecoveryDeviceKey", "RevokeDeviceSessions"},
		"UserProfileService": {"GetCurrentUser", "GetPublicProfile", "BatchGetPublicProfiles", "UpdateProfile"},
		"UserSecurityService": {
			"EnrollMFA",
			"ConfirmMFA",
			"DisableMFA",
			"GenerateRecoveryCodes",
			"ListSecurityEvents",
			"UnlockUser",
			"ListAdminActions",
			"ListSecurityNotices",
			"ResendSecurityNotice",
			"GetSecurityState",
		},
		"UserPolicyService": {"GetUserAppPolicy", "UpdateUserAppPolicy"},
		"RecoveryService": {
			"SubmitRecoveryRequest",
			"CreateTrustedDeviceRecoveryChallenge",
			"VerifyTrustedDeviceRecoveryChallenge",
			"SubmitPaymentRecoveryEvidence",
			"ListRecoveryRequests",
			"GetRecoveryRequest",
			"AcceptRecoveryRequest",
			"ApproveRecoveryRequest",
			"RejectRecoveryRequest",
			"ClaimRecoveryGrant",
			"CompleteRecoveryRequest",
		},
		"UserExportService":    {"CreateUserExportJob", "ListUserExportJobs", "GetUserExportDownload"},
		"UserLifecycleService": {
			"ListLifecycleOperations", "BeginUserDeletion", "CancelUserDeletion", "FinalizeUserDeletion",
		},
	},
	"wallet.v1": {
		"CoinRewardService":    {"PreflightCoinReward", "GrantCoinReward"},
		"WalletClosureService": {"PrepareClosure", "ApplyClosure", "CancelClosure", "GetClosureStatus"},
		"WalletService": {
			"ListWallets", "GetWallet", "ListTransactions", "Credit", "Debit", "Freeze", "Unfreeze",
			"ListFreezes", "TransferToPlatform", "GetPlatformTransferByBusinessKey", "ReverseTransaction",
			"ListWalletEvents", "ListWalletUserDebts", "GetUserWalletSummary", "CreateExport", "ListExports", "RetryExport", "GetExportDownload",
		},
		"WalletRechargeService": {
			"ListConsumerRechargeOffers", "CreateOrGetConsumerRechargeOrder", "PrepareStoreRechargePurchase", "GetRechargeOrderForPayment",
			"ListRechargeProducts", "ListRechargeProductRevisions", "CreateRechargeProductRevision", "ApproveRechargeProductRevision", "CreateRechargeOrder",
			"ListRechargeOrders", "GetRechargeOrder", "CloseRechargeOrder", "CreditRecharge", "MarkRechargeCreditFailed",
			"PrepareRechargeRefund", "ConfirmRechargeRefund", "ApplyRechargeRefund", "CancelRechargeRefund",
		},
		"WalletAdjustmentService": {
			"CreateAdjustment",
			"ApproveAdjustment",
			"RejectAdjustment",
			"WithdrawAdjustment",
			"ListAdjustments",
		},
		"WalletDebtWriteOffService": {"CreateWriteOff", "ApproveWriteOff", "RejectWriteOff", "ListWriteOffs"},
		"WalletInterventionService": {
			"CreateIntervention",
			"ApproveIntervention",
			"RejectIntervention",
			"ListInterventions",
		},
		"WalletRiskService": {
			"ApproveRiskRuleRevision",
			"CreateEmergencyWalletBlock",
			"CreateRiskRuleRevision",
			"EndEmergencyWalletBlock",
			"ListEmergencyWalletBlocks",
			"ListRiskRuleRevisions",
			"ListRiskRules",
		},
		"WalletReconciliationService": {
			"CreateReconciliationBatch",
			"RetryReconciliationBatch",
			"ResolveReconciliationItem",
			"ListReconciliationBatches",
			"ListReconciliationItems",
		},
		"WalletOperationsQueryService": {"GetOperationsSummary"},
	},
	"payment.v1": {
		"PaymentClosureService": {"PrepareClosure", "ApplyClosure", "CancelClosure", "GetClosureStatus"},
		"PaymentService": {
			"ListSubscriptionOffers", "StartFixedTermSubscriptionCheckout", "StartRechargeCheckout", "PrepareStoreRechargePurchase", "VerifyStoreRechargePurchase", "ConfirmStoreRechargePurchaseCompletion", "PrepareAppleSubscriptionPurchase", "VerifyAppleSubscriptionPurchase", "GetAppleSubscriptionManagementAction", "PrepareGoogleSubscriptionPurchase", "VerifyGoogleSubscriptionPurchase", "GetGoogleSubscriptionManagementAction", "RestoreStorePurchases", "GetMyPurchase", "ListMyPurchases", "GetMySubscription", "RetryMyPurchase", "ListPurchaseOptions", "CreatePayment", "GetPayment", "GetPaymentDetail", "GetUserPaymentSummary", "ClosePayment", "SimulatePaymentSuccess", "ListPayments",
			"ListPaymentEvents", "ListPaymentCallbacks", "HandleProviderCallback", "RefreshPaymentProviderState", "ReprocessPaymentCallback",
			"GetPaymentOperationsSummary", "ListPaymentWorkQueue", "ClaimPaymentWorkQueueItem", "ReassignPaymentWorkQueueItem", "CompletePaymentWorkQueueItem",
			"CreatePaymentRiskPolicyRevision", "PublishPaymentRiskPolicyRevision", "ListPaymentRiskPolicyRevisions", "GetPaymentRiskDecision", "ListPaymentRiskReviews", "DecidePaymentRiskReview", "GetPaymentAnalytics",
		},
		"PaymentChannelService": {
			"CreatePaymentChannel",
			"CreateChannelRevision",
			"ProbeChannelRevision",
			"ListPaymentChannels",
			"ListChannelRevisions",
			"CreateRoutingPolicyRevision",
			"CreatePaymentOptionRevision",
			"SubmitChannelPublication",
			"ReviewChannelPublication",
			"ListChannelPublications",
			"ListRoutingPolicyRevisions",
			"PreviewPaymentRouting",
			"ListProviderCapabilitySupport",
		},
		"PaymentRefundService": {
			"CreateRefund",
			"ListRefunds",
			"SubmitRefundRequest",
			"ApproveRefundRequest",
			"RejectRefundRequest",
			"ListRefundRequests",
			"UpsertRefundRiskRule",
			"ListRefundRiskRules",
			"RefreshRefundProviderState",
		},
		"PaymentSubscriptionService": {
			"CreateSubscription",
			"ListUserSubscriptions",
			"GetUserSubscriptionDetail",
			"ListSubscriptionPlans",
			"CreateSubscriptionPlan",
			"UpdateSubscriptionPlan",
			"ListSubscriptionPlanRevisions",
			"CreateSubscriptionPlanRevision",
			"PublishSubscriptionPlanRevision",
			"SubmitSubscriptionAdjustment",
			"ApproveSubscriptionAdjustment",
			"RejectSubscriptionAdjustment",
			"ListSubscriptionAdjustments",
			"UpsertSubscriptionAdjustmentRiskRule",
			"ListSubscriptionAdjustmentRiskRules",
			"ListProviderSubscriptions",
			"GetProviderSubscriptionDetail",
			"VerifyProviderSubscriptionPurchase",
			"ProcessProviderSubscriptionCallback",
			"CancelProviderSubscription",
			"RefreshProviderSubscription",
			"GetProviderSubscriptionAnalytics",
		},
		"PaymentReconciliationService": {
			"CreateReconciliationBatch",
			"GetReconciliationBatch",
			"GetReconciliationFinding",
			"ImportProviderBill",
			"ListReconciliationBatches",
			"ListReconciliationFindings",
			"ListReconciliationItems",
			"MutateReconciliationFinding",
		},
		"PaymentFinancialOperationsService": {
			"ImportSettlementStatement",
			"ListSettlementStatements",
			"GetSettlementStatement",
			"MutateSettlementDiscrepancy",
			"RecordProviderDispute",
			"ListDisputes",
			"GetDispute",
			"RefreshDispute",
			"AddDisputeEvidence",
			"RemoveDisputeEvidence",
			"SubmitDisputeEvidence",
			"ApplyChargeback",
		},
		"PaymentRechargeFulfillmentService": {
			"ListRechargeDeadLetters",
			"ReplayRechargeDeadLetter",
			"ListDeliveryFailures",
			"ReplayRechargeDeliveryFailure",
			"ReplayRefundDeliveryFailure",
			"ReplayFileReferenceDeliveryFailure",
			"ReplayProviderBillDeliveryFailure",
			"ReplayPaymentAttemptExpirationDeliveryFailure",
		},
		"PaymentGovernanceService": {"ListGovernanceEvents", "GetRawDiagnosticData", "EraseDiagnosticData"},
		"PaymentExportService": {
			"CreatePaymentExportJob",
			"GetPaymentExportJob",
			"ListPaymentExportJobs",
			"CreatePaymentExportDownload",
		},
		"PaymentRecoveryEvidenceService": {"VerifyRecoveryPaymentEvidence"},
		"PaymentOperationsQueryService":  {"GetRechargeDeadLetterSummary"},
	},
	"gift.v1": {
		"GiftRewardService":  {"PreflightBackpackGiftReward", "GrantBackpackGiftReward"},
		"GiftClosureService": {"PrepareClosure", "ApplyClosure", "CancelClosure", "GetClosureStatus"},
		"GiftCatalogService": {
			"ListGifts", "GetGift", "CreateGift", "UpdateGiftDraft", "DiscardGiftDraft", "PublishGiftDraft", "CancelScheduledGiftRevision", "EmergencyOfflineGift", "GetGiftArchivePreflight", "ArchiveGift", "ListGiftRevisions", "CompareGiftRevisions", "ApplyCatalogImport",
			"ListGiftCategories", "CreateGiftCategory", "UpdateGiftCategory", "DisableGiftCategory", "ListGiftOperatorAudits",
			"ListGiftSceneTypes", "CreateGiftSceneType", "UpdateGiftSceneType", "DeleteGiftSceneType", "GetGiftSceneTypeImpact", "SetGiftSceneTypeEnabled",
		},
		"GiftSendService": {
			"SendGift",
			"GetGiftSendRecord",
			"GetGiftSendWorkbenchDetail",
			"ReconcileGiftSend",
			"SendBackpackGift",
			"ListGiftSendRecords",
		},
		"GiftBackpackService": {
			"GrantBackpackGift",
			"RevokeBackpackGift",
			"ExpireBackpackItems",
			"AdjustBackpackGift",
			"ListBackpackBalances",
			"ListBackpackBatches",
			"ListBackpackTransactions",
			"ListBackpackTransactionItems",
		},
		"GiftSettlementService": {
			"ListGiftSettlementRecords", "ListGiftReconciliationIssues", "GetGiftReconciliationIssue", "GetGiftReconciliationIssueAuditDetails",
			"AcknowledgeGiftReconciliationIssue", "AddGiftReconciliationIssueNote", "ReconcileGiftSettlement",
		},
		"GiftAnalyticsService": {"GetGiftOverview", "GetUserGiftSummary", "ListGiftEvents", "ListGiftStats"},
		"GiftExportService": {
			"CreateGiftExportJob",
			"ListGiftExportJobs",
			"GetGiftExportJob",
			"CreateGiftExportDownloadGrant",
			"RedeemGiftExportDownloadGrant",
		},
	},
	"support.verification.v1": {
		"VerificationService": {
			"SendCode",
			"VerifyCode",
			"ConsumeVerificationTicket",
			"ListVerificationAudits",
			"GetVerificationAudit",
			"RevealVerificationRecipientPhone",
		},
	},
	"support.messaging.v1": {
		"SmsService": {
			"SendTemplateSms", "GetSmsMessageByIdempotency", "ListSmsMessages",
			"ListSmsDeliveryAttempts", "RevealSmsRecipientPhone",
		},
	},
	"support.file.v1": {
		"FileUploadService": {
			"CreateUploadSession",
			"CompleteUploadSession",
			"AbortUploadSession",
			"GetUploadSession",
			"StoreServiceFile",
		},
		"FileAccessService": {
			"GetFileView",
			"BatchGetFileViews",
			"ValidateFile",
			"BatchValidateFiles",
			"CreateDownloadAuthorization",
			"IssueProtectedDownload",
		},
		"FileReferenceService": {"AttachAndPublish", "DetachReference"},
		"FileAdministrationService": {
			"GetFile", "ListFiles", "ListUploadSessions", "PublishStandaloneFile", "RevokeFile",
			"ListFileReferences", "ListFileEvents", "ListFileReviewAttempts", "CreateFilePolicyDraft", "PreflightFilePolicyDraft", "ActivateFilePolicyDraft", "DiscardFilePolicyDraft", "ListFilePolicyRevisions",
			"CreateStorageProviderDraft", "PreflightStorageProviderDraft", "ActivateStorageProviderDraft", "DiscardStorageProviderDraft", "ListStorageProviderRevisions",
			"CreateReconciliationBatch", "ListReconciliationBatches", "ListReconciliationItems", "ListFileTasks", "RetryFileTask",
		},
	},
	"support.governance.v1": {
		"GovernanceService": {"ListGovernanceEvents"},
	},
}

var adminContracts = serviceContract{
	"WalletOperationsService": {
		"GetOverview",
		"GetRechargeOrderDetail",
		"GetUserWorkbench",
		"ListUserWalletTransactions",
		"ListUserWalletEvents",
		"ListUserFreezes",
		"ListUserDebts",
		"ListUserAdjustments",
		"ListUserRechargeOrders",
		"ListUserPayments",
		"ListUserRefunds",
		"ListUserGiftSends",
		"CreateExport",
		"ListExports",
		"RetryExport",
		"GetExportDownload",
	},
	"WalletService": {
		"ListWallets",
		"GetWallet",
		"ListTransactions",
		"Freeze",
		"ListFreezes",
		"ListWalletEvents",
		"ListWalletUserDebts",
	},
	"WalletRechargeService": {
		"ListRechargeProducts",
		"ListRechargeProductRevisions",
		"CreateRechargeProductRevision",
		"ApproveRechargeProductRevision",
		"ListRechargeOrders",
		"CloseRechargeOrder",
	},
	"WalletAdjustmentService": {
		"CreateAdjustment",
		"ApproveAdjustment",
		"RejectAdjustment",
		"WithdrawAdjustment",
		"ListAdjustments",
	},
	"WalletDebtWriteOffService": {"CreateWriteOff", "ApproveWriteOff", "RejectWriteOff", "ListWriteOffs"},
	"WalletInterventionService": {
		"CreateIntervention",
		"ApproveIntervention",
		"RejectIntervention",
		"ListInterventions",
	},
	"WalletRiskService": {
		"ApproveRiskRuleRevision",
		"CreateEmergencyWalletBlock",
		"CreateRiskRuleRevision",
		"EndEmergencyWalletBlock",
		"ListEmergencyWalletBlocks",
		"ListRiskRuleRevisions",
		"ListRiskRules",
	},
	"WalletReconciliationService": {
		"CreateReconciliationBatch",
		"RetryReconciliationBatch",
		"ResolveReconciliationItem",
		"ListReconciliationBatches",
		"ListReconciliationItems",
	},
	"PaymentService": {
		"GetPayment",
		"GetUserPaymentSummary",
		"ClosePayment",
		"RefreshPaymentProviderState",
		"ListPayments",
		"ListPaymentEvents",
		"ListPaymentCallbacks",
		"ReprocessPaymentCallback",
		"GetPaymentOperationsSummary",
		"ListPaymentWorkQueue",
		"ClaimRefundRequestWorkQueueItem",
		"ReassignRefundRequestWorkQueueItem",
		"CompleteRefundRequestWorkQueueItem",
		"ClaimDeliveryFailureWorkQueueItem",
		"ReassignDeliveryFailureWorkQueueItem",
		"CompleteDeliveryFailureWorkQueueItem",
		"ClaimRiskReviewWorkQueueItem",
		"ReassignRiskReviewWorkQueueItem",
		"CompleteRiskReviewWorkQueueItem",
		"CreatePaymentRiskPolicyRevision",
		"PublishPaymentRiskPolicyRevision",
		"ListPaymentRiskPolicyRevisions",
		"GetPaymentRiskDecision",
		"ListPaymentRiskReviews",
		"ApprovePaymentRiskReview",
		"RejectPaymentRiskReview",
		"GetPaymentAnalytics",
	},
	"PaymentChannelService": {
		"CreatePaymentChannel",
		"CreateChannelRevision",
		"ProbeChannelRevision",
		"ListPaymentChannels",
		"ListChannelRevisions",
		"CreateRoutingPolicyRevision",
		"SubmitChannelPublication",
		"ReviewChannelPublication",
		"ListChannelPublications",
		"ListRoutingPolicyRevisions",
		"PreviewPaymentRouting",
		"ListProviderCapabilitySupport",
	},
	"PaymentRefundService": {
		"SubmitRefundRequest",
		"ApproveRefundRequest",
		"RejectRefundRequest",
		"ListRefundRequests",
		"UpsertRefundRiskRule",
		"ListRefundRiskRules",
		"RefreshRefundProviderState",
		"ListRefunds",
	},
	"PaymentTestService": {"CreatePayment", "SimulatePaymentSuccess"},
	"PaymentSubscriptionService": {
		"ListUserSubscriptions",
		"GetUserSubscriptionDetail",
		"ListSubscriptionPlans",
		"CreateSubscriptionPlan",
		"UpdateSubscriptionPlan",
		"ListSubscriptionPlanRevisions",
		"CreateSubscriptionPlanRevision",
		"PublishSubscriptionPlanRevision",
		"SubmitSubscriptionAdjustment",
		"ApproveSubscriptionAdjustment",
		"RejectSubscriptionAdjustment",
		"ListSubscriptionAdjustments",
		"UpsertSubscriptionAdjustmentRiskRule",
		"ListSubscriptionAdjustmentRiskRules",
		"ListProviderSubscriptions",
		"GetProviderSubscriptionDetail",
		"VerifyProviderSubscriptionPurchase",
		"CancelProviderSubscription",
		"RefreshProviderSubscription",
		"GetProviderSubscriptionAnalytics",
	},
	"PaymentReconciliationService": {
		"CreateReconciliationBatch",
		"GetReconciliationBatch",
		"GetReconciliationFinding",
		"ImportProviderBill",
		"ListReconciliationBatches",
		"ListReconciliationFindings",
		"ListReconciliationItems",
		"MutateReconciliationFinding",
	},
	"PaymentFinancialOperationsService": {
		"ImportSettlementStatement",
		"ListSettlementStatements",
		"GetSettlementStatement",
		"MutateSettlementDiscrepancy",
		"ListDisputes",
		"GetDispute",
		"RefreshDispute",
		"AddDisputeEvidence",
		"RemoveDisputeEvidence",
		"SubmitDisputeEvidence",
		"ApplyChargeback",
	},
	"PaymentRechargeFulfillmentService": {
		"ListRechargeDeadLetters",
		"ReplayRechargeDeadLetter",
		"ListDeliveryFailures",
		"ReplayRechargeDeliveryFailure",
		"ReplayRefundDeliveryFailure",
		"ReplayFileReferenceDeliveryFailure",
		"ReplayProviderBillDeliveryFailure",
		"ReplayPaymentAttemptExpirationDeliveryFailure",
	},
	"PaymentGovernanceService": {"ListGovernanceEvents", "GetRawDiagnosticData", "EraseDiagnosticData"},
	"PaymentExportService": {
		"CreatePaymentExportJob",
		"GetPaymentExportJob",
		"ListPaymentExportJobs",
		"CreatePaymentExportDownload",
	},
	"GiftCatalogService": {
		"ListGifts",
		"GetGift",
		"CreateGift",
		"UpdateGiftDraft",
		"DiscardGiftDraft",
		"PublishGiftDraft",
		"CancelScheduledGiftRevision",
		"EmergencyOfflineGift",
		"GetGiftArchivePreflight",
		"ArchiveGift",
		"ListGiftRevisions",
		"CompareGiftRevisions",
		"ApplyCatalogImport",
		"ListGiftCategories",
		"CreateGiftCategory",
		"UpdateGiftCategory",
		"DisableGiftCategory",
		"ListGiftSceneTypes",
		"CreateGiftSceneType",
		"UpdateGiftSceneType",
		"DeleteGiftSceneType",
		"GetGiftSceneTypeImpact",
		"SetGiftSceneTypeEnabled",
		"ListGiftOperatorAudits",
	},
	"GiftSendService": {"GetGiftSendRecord", "ListGiftSendRecords", "ReconcileGiftSend"},
	"GiftBackpackService": {
		"AdjustBackpackGift",
		"ListBackpackBalances",
		"ListBackpackBatches",
		"ListBackpackTransactions",
		"ListBackpackTransactionItems",
	},
	"GiftSettlementService": {
		"ListGiftSettlementRecords",
		"ListGiftReconciliationIssues", "GetGiftReconciliationIssue", "GetGiftReconciliationIssueAuditDetails",
		"AcknowledgeGiftReconciliationIssue", "AddGiftReconciliationIssueNote", "ReconcileGiftSettlement",
	},
	"GiftAnalyticsService": {"GetGiftOverview", "ListGiftStats", "ListGiftEvents"},
	"GiftExportService": {
		"CreateGiftExportJob",
		"ListGiftExportJobs",
		"GetGiftExportJob",
		"CreateGiftExportDownloadGrant",
		"RedeemGiftExportDownloadGrant",
	},
	"SupportVerificationService": {
		"ListVerificationAudits",
		"GetVerificationAudit",
		"RevealVerificationRecipientPhone",
	},
	"SupportSmsService":        {"ListSmsMessages", "ListSmsDeliveryAttempts", "RevealSmsRecipientPhone"},
	"SupportFileUploadService": {"CreateUploadSession", "CompleteUploadSession", "GetUploadSession"},
	"SupportFileAccessService": {"GetFileView", "BatchGetFileViews"},
	"SupportFileAdministrationService": {
		"GetFile", "ListFiles", "ListUploadSessions", "PublishStandaloneFile", "RevokeFile",
		"ListFileReferences", "ListFileEvents", "ListFileReviewAttempts", "CreateFilePolicyDraft", "PreflightFilePolicyDraft", "ActivateFilePolicyDraft", "DiscardFilePolicyDraft", "ListFilePolicyRevisions",
		"CreateStorageProviderDraft", "PreflightStorageProviderDraft", "ActivateStorageProviderDraft", "DiscardStorageProviderDraft", "ListStorageProviderRevisions",
		"CreateReconciliationBatch", "ListReconciliationBatches", "ListReconciliationItems", "ListFileTasks", "RetryFileTask",
	},
	"SupportGovernanceService": {"ListGovernanceEvents"},
}

var expectedAdminRoutes = []string{
	"GET /admin/v1/wallet-operations/overview", "GET /admin/v1/wallet-operations/recharge-orders/{order_no}", "GET /admin/v1/wallet-operations/users/{user_id}",
	"GET /admin/v1/wallet-operations/users/{user_id}/wallet-transactions", "GET /admin/v1/wallet-operations/users/{user_id}/wallet-events", "GET /admin/v1/wallet-operations/users/{user_id}/freezes", "GET /admin/v1/wallet-operations/users/{user_id}/debts",
	"GET /admin/v1/wallet-operations/users/{user_id}/adjustments", "GET /admin/v1/wallet-operations/users/{user_id}/recharge-orders", "GET /admin/v1/wallet-operations/users/{user_id}/payments", "GET /admin/v1/wallet-operations/users/{user_id}/refunds", "GET /admin/v1/wallet-operations/users/{user_id}/gift-sends",
	"POST /admin/v1/wallet-operations/exports", "GET /admin/v1/wallet-operations/exports", "POST /admin/v1/wallet-operations/exports/{export_id}:retry", "POST /admin/v1/wallet-operations/exports/{export_id}:download",
	"GET /admin/v1/wallets", "GET /admin/v1/wallets/{user_id}", "GET /admin/v1/wallet-transactions",
	"POST /admin/v1/wallet-freezes", "GET /admin/v1/wallet-freezes", "GET /admin/v1/wallet-events", "GET /admin/v1/wallet-user-debts",
	"POST /admin/v1/wallet-adjustments", "POST /admin/v1/wallet-adjustments:approve", "POST /admin/v1/wallet-adjustments:reject", "POST /admin/v1/wallet-adjustments:withdraw", "GET /admin/v1/wallet-adjustments",
	"POST /admin/v1/wallet-interventions", "POST /admin/v1/wallet-interventions:approve", "POST /admin/v1/wallet-interventions:reject", "GET /admin/v1/wallet-interventions",
	"POST /admin/v1/wallet-debt-write-offs", "POST /admin/v1/wallet-debt-write-offs:approve", "POST /admin/v1/wallet-debt-write-offs:reject", "GET /admin/v1/wallet-debt-write-offs",
	"GET /admin/v1/wallet-risk-rules", "GET /admin/v1/wallet-risk-rules/{rule_id}/revisions", "POST /admin/v1/wallet-risk-rule-revisions", "POST /admin/v1/wallet-risk-rule-revisions:approve", "GET /admin/v1/emergency-wallet-blocks", "POST /admin/v1/emergency-wallet-blocks", "POST /admin/v1/emergency-wallet-blocks:end",
	"POST /admin/v1/wallet-reconciliation-batches", "POST /admin/v1/wallet-reconciliation-batches/{batch_id}:retry", "GET /admin/v1/wallet-reconciliation-batches", "GET /admin/v1/wallet-reconciliation-items", "POST /admin/v1/wallet-reconciliation-items/{item_id}:resolve",
	"GET /admin/v1/recharge-products", "GET /admin/v1/recharge-products/{product_id}/revisions", "POST /admin/v1/recharge-product-revisions", "POST /admin/v1/recharge-product-revisions:approve", "GET /admin/v1/recharge-orders", "POST /admin/v1/recharge-orders:close",
	"GET /admin/v1/payments/{payment_no}", "POST /admin/v1/payments/{payment_no}:close", "POST /admin/v1/payments/{payment_no}:refresh-provider-state",
	"GET /admin/v1/payments", "GET /admin/v1/payment-events", "GET /admin/v1/payment-callbacks", "POST /admin/v1/payment-callbacks/{callback_id}:reprocess",
	"GET /admin/v1/users/{user_id}/payment-summary",
	"GET /admin/v1/payment-operations/summary", "GET /admin/v1/payment-work-queue",
	"POST /admin/v1/payment-work-queue/refund-requests/{resource_id}:claim", "POST /admin/v1/payment-work-queue/refund-requests/{resource_id}:reassign", "POST /admin/v1/payment-work-queue/refund-requests/{resource_id}:complete",
	"POST /admin/v1/payment-work-queue/delivery-failures/{resource_id}:claim", "POST /admin/v1/payment-work-queue/delivery-failures/{resource_id}:reassign", "POST /admin/v1/payment-work-queue/delivery-failures/{resource_id}:complete",
	"POST /admin/v1/payment-work-queue/risk-reviews/{resource_id}:claim", "POST /admin/v1/payment-work-queue/risk-reviews/{resource_id}:reassign", "POST /admin/v1/payment-work-queue/risk-reviews/{resource_id}:complete",
	"POST /admin/v1/payment-risk-policy-revisions", "GET /admin/v1/payment-risk-policy-revisions", "POST /admin/v1/payment-risk-policy-revisions/{revision_id}:publish",
	"GET /admin/v1/payments/{payment_no}/risk-decision", "GET /admin/v1/payment-risk-reviews", "POST /admin/v1/payment-risk-reviews/{review_id}:approve", "POST /admin/v1/payment-risk-reviews/{review_id}:reject", "GET /admin/v1/payment-analytics",
	"POST /admin/v1/payment-exports", "GET /admin/v1/payment-exports", "GET /admin/v1/payment-exports/{job_no}", "POST /admin/v1/payment-exports/{job_no}:download",
	"POST /admin/v1/payment-test/payments", "POST /admin/v1/payment-test/payments/{payment_no}:simulate-success",
	"GET /admin/v1/payment-recharge-dead-letters", "POST /admin/v1/payment-recharge-dead-letters/{event_id}:replay",
	"GET /admin/v1/payment-delivery-failures", "POST /admin/v1/payment-delivery-failures/recharge/{event_id}:replay", "POST /admin/v1/payment-delivery-failures/refund/{event_id}:replay", "POST /admin/v1/payment-delivery-failures/file-reference/{event_id}:replay", "POST /admin/v1/payment-delivery-failures/provider-bill/{event_id}:replay", "POST /admin/v1/payment-delivery-failures/payment-attempt-expiration/{event_id}:replay",
	"GET /admin/v1/payment-governance-events", "GET /admin/v1/payment-diagnostics/{resource_type}/{resource_id}:raw", "POST /admin/v1/payment-diagnostics/{resource_type}/{resource_id}:erase",
	"POST /admin/v1/payment-channels", "GET /admin/v1/payment-channels", "POST /admin/v1/payment-channels/{channel_id}/revisions", "GET /admin/v1/payment-channels/{channel_id}/revisions", "POST /admin/v1/payment-channel-revisions/{revision_id}:probe", "GET /admin/v1/payment-provider-capabilities",
	"POST /admin/v1/payment-routing-policy-revisions", "GET /admin/v1/payment-routing-policy-revisions", "POST /admin/v1/payment-channel-publications", "GET /admin/v1/payment-channel-publications", "POST /admin/v1/payment-channel-publications/{publication_id}:review", "POST /admin/v1/payment-routing:preview",
	"POST /admin/v1/refund-requests", "POST /admin/v1/refund-requests/{operation_no}:approve", "POST /admin/v1/refund-requests/{operation_no}:reject", "GET /admin/v1/refund-requests",
	"PUT /admin/v1/refund-risk-rules/{currency}", "GET /admin/v1/refund-risk-rules", "GET /admin/v1/refunds", "POST /admin/v1/refunds/{refund_no}:refresh-provider-state",
	"POST /admin/v1/payment/provider-bills:import", "POST /admin/v1/payment/reconciliation-batches", "GET /admin/v1/payment/reconciliation-batches", "GET /admin/v1/payment/reconciliation-batches/{batch_id}",
	"GET /admin/v1/payment/reconciliation-findings", "GET /admin/v1/payment/reconciliation-findings/{finding_id}", "POST /admin/v1/payment/reconciliation-findings/{finding_id}:mutate", "GET /admin/v1/reconciliation-items",
	"POST /admin/v1/payment/settlements:import", "GET /admin/v1/payment/settlements", "GET /admin/v1/payment/settlements/{statement_id}", "POST /admin/v1/payment/settlement-discrepancies/{discrepancy_id}:mutate",
	"GET /admin/v1/payment/disputes", "GET /admin/v1/payment/disputes/{dispute_id}", "POST /admin/v1/payment/disputes/{dispute_id}:refresh", "POST /admin/v1/payment/disputes/{dispute_id}/evidences", "POST /admin/v1/payment/disputes/{dispute_id}/evidences/{evidence_id}:remove", "POST /admin/v1/payment/disputes/{dispute_id}:submit-evidence", "POST /admin/v1/payment/disputes/{dispute_id}:apply-chargeback",
	"GET /admin/v1/user-subscriptions", "GET /admin/v1/user-subscriptions/{id}",
	"GET /admin/v1/subscription-plans", "POST /admin/v1/subscription-plans", "PUT /admin/v1/subscription-plans/{id}",
	"GET /admin/v1/subscription-plans/{plan_id}/revisions", "POST /admin/v1/subscription-plans/{plan_id}/revisions", "POST /admin/v1/subscription-plan-revisions/{revision_id}:publish",
	"GET /admin/v1/subscription-adjustments", "POST /admin/v1/subscription-adjustments", "POST /admin/v1/subscription-adjustments/{operation_no}:approve", "POST /admin/v1/subscription-adjustments/{operation_no}:reject",
	"GET /admin/v1/subscription-adjustment-risk-rules", "PUT /admin/v1/subscription-adjustment-risk-rules/{currency}",
	"GET /admin/v1/provider-subscriptions", "GET /admin/v1/provider-subscriptions/{id}", "GET /admin/v1/provider-subscriptions:analytics",
	"POST /admin/v1/provider-subscriptions:verify-purchase", "POST /admin/v1/provider-subscriptions/{id}:cancel", "POST /admin/v1/provider-subscriptions/{id}:refresh",
	"GET /admin/v1/gifts", "GET /admin/v1/gifts/{id}", "POST /admin/v1/gifts", "PUT /admin/v1/gifts/{id}/draft", "DELETE /admin/v1/gifts/{id}/draft", "POST /admin/v1/gifts/{id}/draft:publish", "POST /admin/v1/gifts/{id}/scheduled-revision:cancel", "POST /admin/v1/gifts/{id}:emergency-offline", "GET /admin/v1/gifts/{id}/archive-preflight", "POST /admin/v1/gifts/{id}:archive", "GET /admin/v1/gifts/{id}/revisions", "GET /admin/v1/gifts/{gift_id}/revisions:diff", "POST /admin/v1/gifts:import",
	"GET /admin/v1/gift-categories", "POST /admin/v1/gift-categories", "PUT /admin/v1/gift-categories/{id}", "POST /admin/v1/gift-categories/{id}:disable", "GET /admin/v1/gift-operator-audits",
	"GET /admin/v1/gift-scene-types", "POST /admin/v1/gift-scene-types", "PUT /admin/v1/gift-scene-types/{id}", "DELETE /admin/v1/gift-scene-types/{id}", "GET /admin/v1/gift-scene-types/{id}/impact", "POST /admin/v1/gift-scene-types/{id}:set-enabled",
	"GET /admin/v1/gift-send-records/{id}", "POST /admin/v1/gift-send-records/{send_id}:reconcile",
	"POST /admin/v1/gift-backpacks:adjust",
	"GET /admin/v1/gift-backpacks", "GET /admin/v1/gift-backpack-batches", "GET /admin/v1/gift-backpack-transactions", "GET /admin/v1/gift-backpack-transactions/{transaction_id}/items",
	"GET /admin/v1/gift-send-records", "GET /admin/v1/gift-overview",
	"GET /admin/v1/gift-settlement-records", "GET /admin/v1/gift-stats", "GET /admin/v1/gift-events",
	"GET /admin/v1/gift-reconciliation-issues", "GET /admin/v1/gift-reconciliation-issues/{id}", "GET /admin/v1/gift-reconciliation-issues/{id}/audit-details",
	"POST /admin/v1/gift-reconciliation-issues/{id}:acknowledge", "POST /admin/v1/gift-reconciliation-issues/{id}:add-note", "POST /admin/v1/gift-reconciliation-issues/{issue_id}:reconcile",
	"POST /admin/v1/gift-export-jobs", "GET /admin/v1/gift-export-jobs", "GET /admin/v1/gift-export-jobs/{id}", "POST /admin/v1/gift-export-jobs/{job_id}/download-grants", "POST /admin/v1/gift-export-download-grants:redeem",
	"GET /admin/v1/support/verification-audits", "GET /admin/v1/support/verification-audits/{audit_id}", "POST /admin/v1/support/verification-audits/{audit_id}:reveal-recipient-phone",
	"GET /admin/v1/support/sms-messages", "GET /admin/v1/support/sms-delivery-attempts", "POST /admin/v1/support/sms-messages/{message_no}:reveal-recipient-phone",
	"GET /admin/v1/support/governance-events",
	"POST /admin/v1/file-upload-sessions", "POST /admin/v1/file-upload-sessions/{session_no}:complete", "GET /admin/v1/file-upload-sessions/{session_no}",
	"GET /admin/v1/files/{file_id}/view", "POST /admin/v1/files:batch-get-views",
	"GET /admin/v1/files/{id}", "GET /admin/v1/files", "GET /admin/v1/file-upload-sessions",
	"POST /admin/v1/files/{file_id}:publish-standalone", "POST /admin/v1/files/{file_id}:revoke",
	"GET /admin/v1/file-references", "GET /admin/v1/file-events", "GET /admin/v1/file-review-attempts",
	"POST /admin/v1/file-policy-revisions", "POST /admin/v1/file-policy-revisions/{revision_id}:preflight", "POST /admin/v1/file-policy-revisions/{revision_id}:activate", "POST /admin/v1/file-policy-revisions/{revision_id}:discard", "GET /admin/v1/file-policy-revisions",
	"POST /admin/v1/file-provider-revisions", "POST /admin/v1/file-provider-revisions/{revision_id}:preflight", "POST /admin/v1/file-provider-revisions/{revision_id}:activate", "POST /admin/v1/file-provider-revisions/{revision_id}:discard", "GET /admin/v1/file-provider-revisions",
	"POST /admin/v1/file-reconciliation-batches", "GET /admin/v1/file-reconciliation-batches", "GET /admin/v1/file-reconciliation-items",
	"GET /admin/v1/file-tasks", "POST /admin/v1/file-tasks/{task_id}:retry",
}

type parsedService struct {
	rpcs   []string
	routes []string
}

// main 作为 main 的进程入口，负责在运行边界前完成检查或装配。
func main() {
	root := flag.String("root", ".", "Proto root to inspect")
	flag.Parse()

	packages, err := parseProtoTree(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var failures []string
	for pkg, expected := range businessContracts {
		failures = append(failures, compareContracts(pkg, packages[pkg], expected)...)
	}
	adminServices := make(map[string]parsedService)
	for packageName, services := range packages {
		if !strings.HasPrefix(packageName, "admin.") {
			continue
		}
		for name, service := range services {
			if _, expected := adminContracts[name]; expected || name == "GiftService" || name == "RechargeService" {
				adminServices[name] = service
			}
		}
	}
	failures = append(failures, compareContracts("admin functional packages", adminServices, adminContracts)...)

	var adminRoutes []string
	for service, parsed := range adminServices {
		if service == "GiftService" || service == "RechargeService" {
			adminRoutes = append(adminRoutes, parsed.routes...)
			continue
		}
		if _, expected := adminContracts[service]; expected {
			adminRoutes = append(adminRoutes, parsed.routes...)
		}
	}
	sort.Strings(adminRoutes)
	sort.Strings(expectedAdminRoutes)
	if !equalStrings(adminRoutes, expectedAdminRoutes) {
		failures = append(
			failures,
			fmt.Sprintf(
				"admin REST routes changed:\n  got:  %s\n  want: %s",
				strings.Join(adminRoutes, ", "),
				strings.Join(expectedAdminRoutes, ", "),
			),
		)
	}

	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Fprintln(os.Stderr, "domain service contract violation:", failure)
		}
		os.Exit(1)
	}
	fmt.Println("domain service contracts passed")
}

// parseProtoTree 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func parseProtoTree(root string) (map[string]map[string]parsedService, error) {
	result := make(map[string]map[string]parsedService)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".proto" {
			return walkErr
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		match := packagePattern.FindSubmatch(contents)
		if len(match) == 0 {
			return nil
		}
		pkg := string(match[1])
		for offset := 0; ; {
			serviceMatch := servicePattern.FindSubmatchIndex(contents[offset:])
			if serviceMatch == nil {
				break
			}
			name := string(contents[offset+serviceMatch[2] : offset+serviceMatch[3]])
			open := offset + serviceMatch[1] - 1
			close, err := matchingBrace(contents, open)
			if err != nil {
				return fmt.Errorf("parse %s service %s: %w", path, name, err)
			}
			body := contents[open+1 : close]
			parsed := parsedService{}
			for _, rpcMatch := range rpcPattern.FindAllSubmatch(body, -1) {
				parsed.rpcs = append(parsed.rpcs, string(rpcMatch[1]))
			}
			for _, routeMatch := range httpPattern.FindAllSubmatch(body, -1) {
				parsed.routes = append(parsed.routes, strings.ToUpper(string(routeMatch[1]))+" "+string(routeMatch[2]))
			}
			if result[pkg] == nil {
				result[pkg] = make(map[string]parsedService)
			}
			if _, exists := result[pkg][name]; exists {
				return fmt.Errorf("service %s.%s is declared more than once", pkg, name)
			}
			result[pkg][name] = parsed
			offset = close + 1
		}
		return nil
	})
	return result, err
}

// matchingBrace 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func matchingBrace(contents []byte, open int) (int, error) {
	depth := 0
	inString := false
	escaped := false
	for index := open; index < len(contents); index++ {
		value := contents[index]
		if inString {
			if value == '"' && !escaped {
				inString = false
			}
			escaped = value == '\\' && !escaped
			if value != '\\' {
				escaped = false
			}
			continue
		}
		if value == '"' {
			inString = true
			continue
		}
		switch value {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index, nil
			}
		}
	}
	return 0, fmt.Errorf("unclosed service body")
}

// compareContracts 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func compareContracts(label string, actual map[string]parsedService, expected serviceContract) []string {
	var failures []string
	actualNames := make([]string, 0, len(actual))
	for name := range actual {
		actualNames = append(actualNames, name)
	}
	expectedNames := make([]string, 0, len(expected))
	for name := range expected {
		expectedNames = append(expectedNames, name)
	}
	sort.Strings(actualNames)
	sort.Strings(expectedNames)
	if !equalStrings(actualNames, expectedNames) {
		failures = append(
			failures,
			fmt.Sprintf(
				"%s services got [%s], want [%s]",
				label,
				strings.Join(actualNames, ", "),
				strings.Join(expectedNames, ", "),
			),
		)
	}
	for service, want := range expected {
		got := append([]string(nil), actual[service].rpcs...)
		sort.Strings(got)
		want = append([]string(nil), want...)
		sort.Strings(want)
		if !equalStrings(got, want) {
			failures = append(
				failures,
				fmt.Sprintf(
					"%s.%s RPCs got [%s], want [%s]",
					label,
					service,
					strings.Join(got, ", "),
					strings.Join(want, ", "),
				),
			)
		}
	}
	return failures
}

// equalStrings 完成 main 的内部协作步骤，保持调用链既有的输入校验、错误传播和状态约束。
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
