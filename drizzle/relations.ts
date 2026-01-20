import { relations } from "drizzle-orm/relations";
import { fixedIncomeProduct, subscriptionOrder, users, lendingPositions, withdrawalAddresses, addressVerifications, withdrawals, deposits, walletCreationRequests, withdrawalVerification } from "./schema";

export const subscriptionOrderRelations = relations(subscriptionOrder, ({one}) => ({
	fixedIncomeProduct: one(fixedIncomeProduct, {
		fields: [subscriptionOrder.productId],
		references: [fixedIncomeProduct.id]
	}),
}));

export const fixedIncomeProductRelations = relations(fixedIncomeProduct, ({many}) => ({
	subscriptionOrders: many(subscriptionOrder),
}));

export const lendingPositionsRelations = relations(lendingPositions, ({one}) => ({
	user: one(users, {
		fields: [lendingPositions.userId],
		references: [users.id]
	}),
}));

export const usersRelations = relations(users, ({many}) => ({
	lendingPositions: many(lendingPositions),
	withdrawalAddresses: many(withdrawalAddresses),
	withdrawals: many(withdrawals),
	deposits: many(deposits),
	walletCreationRequests: many(walletCreationRequests),
	withdrawalVerifications: many(withdrawalVerification),
}));

export const withdrawalAddressesRelations = relations(withdrawalAddresses, ({one, many}) => ({
	user: one(users, {
		fields: [withdrawalAddresses.userId],
		references: [users.id]
	}),
	addressVerifications: many(addressVerifications),
	withdrawals: many(withdrawals),
}));

export const addressVerificationsRelations = relations(addressVerifications, ({one}) => ({
	withdrawalAddress: one(withdrawalAddresses, {
		fields: [addressVerifications.addressId],
		references: [withdrawalAddresses.id]
	}),
}));

export const withdrawalsRelations = relations(withdrawals, ({one}) => ({
	user: one(users, {
		fields: [withdrawals.userId],
		references: [users.id]
	}),
	withdrawalAddress: one(withdrawalAddresses, {
		fields: [withdrawals.fromAddressId],
		references: [withdrawalAddresses.id]
	}),
}));

export const depositsRelations = relations(deposits, ({one}) => ({
	user: one(users, {
		fields: [deposits.userId],
		references: [users.id]
	}),
}));

export const walletCreationRequestsRelations = relations(walletCreationRequests, ({one}) => ({
	user: one(users, {
		fields: [walletCreationRequests.userId],
		references: [users.id]
	}),
}));

export const withdrawalVerificationRelations = relations(withdrawalVerification, ({one}) => ({
	user: one(users, {
		fields: [withdrawalVerification.userId],
		references: [users.id]
	}),
}));