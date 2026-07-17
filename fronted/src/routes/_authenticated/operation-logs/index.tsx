import { createFileRoute } from '@tanstack/react-router'
import { requireAdmin } from '@/features/auth/require-admin'
import { OperationLogsPage } from '@/features/order-inventory/operation-logs'

export const Route = createFileRoute('/_authenticated/operation-logs/')({
  beforeLoad: ({ context }) => requireAdmin(context.queryClient),
  component: OperationLogsPage,
})
