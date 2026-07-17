import { useQuery } from '@tanstack/react-query'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { operationLogApi, queryKeys } from './api'
import { ApiErrorPanel, BusinessPage, EmptyRow, LoadingRow } from './components'
import { formatDateTime } from './format'

const PAGE_SIZE = 20

export function OperationLogsPage() {
  const page = 1
  const operationLogsQuery = useQuery({
    queryKey: queryKeys.operationLogs(page, PAGE_SIZE),
    queryFn: () => operationLogApi.list(page, PAGE_SIZE),
  })

  const operationLogs = operationLogsQuery.data?.operation_logs ?? []

  return (
    <BusinessPage
      title='操作日志'
      description='查看管理员后台操作审计记录。'
    >
      <Card>
        <CardHeader>
          <CardTitle>审计列表</CardTitle>
          <CardDescription>记录管理员、接口动作、请求结果和请求 ID。</CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <ApiErrorPanel error={operationLogsQuery.error} />
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>管理员</TableHead>
                <TableHead>动作</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>路径</TableHead>
                <TableHead>请求 ID</TableHead>
                <TableHead>IP</TableHead>
                <TableHead>时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {operationLogsQuery.isLoading && <LoadingRow colSpan={8} />}
              {!operationLogsQuery.isLoading &&
                operationLogs.map((log) => (
                  <TableRow key={log.id}>
                    <TableCell>#{log.id}</TableCell>
                    <TableCell>
                      {log.username || '-'} #{log.user_id}
                    </TableCell>
                    <TableCell className='font-mono text-xs'>
                      {log.action}
                    </TableCell>
                    <TableCell>{log.http_status}</TableCell>
                    <TableCell className='max-w-[240px] whitespace-normal font-mono text-xs'>
                      {log.path}
                    </TableCell>
                    <TableCell className='max-w-[220px] truncate font-mono text-xs'>
                      {log.request_id || '-'}
                    </TableCell>
                    <TableCell>{log.client_ip || '-'}</TableCell>
                    <TableCell>{formatDateTime(log.created_at)}</TableCell>
                  </TableRow>
                ))}
              {!operationLogsQuery.isLoading && operationLogs.length === 0 && (
                <EmptyRow colSpan={8} message='暂无操作日志' />
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </BusinessPage>
  )
}
