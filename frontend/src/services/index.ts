import { AppService, ProviderService, StatsService } from "../../bindings/github.com/wanglejiu/llm-proxy";

// Provider API
export const ProviderAPI = {
  getProviders: () => ProviderService.GetProviders().then((r: any) => r || []),
  createProvider: (data: any) => ProviderService.CreateProvider(data),
  updateProvider: (id: number, data: any) => ProviderService.UpdateProvider(id, data),
  deleteProvider: (id: number) => ProviderService.DeleteProvider(id),
  fetchModels: (baseURL: string, apiKey: string) =>
    ProviderService.FetchProviderModels(baseURL, apiKey).then((r: any) => r || []),
  testConnection: (baseURL: string, apiKey: string, model: string, urlSuffix: string, autoSuffix: boolean) =>
    ProviderService.TestProviderConnection(baseURL, apiKey, model, urlSuffix, autoSuffix),
};

// Stats API
export const StatsAPI = {
  getStats: (providerID = 0) => StatsService.GetStats(providerID),
  getDailyStats: (providerID = 0) => StatsService.GetDailyStats(providerID).then((r: any) => r || []),
  getHourlyStatsByDate: (date: string, providerID = 0) => StatsService.GetHourlyStatsByDate(date, providerID).then((r: any) => r || []),
  getHourlyStatsByDateWithBreakdown: (date: string, providerID = 0) => StatsService.GetHourlyStatsByDateWithBreakdown(date, providerID).then((r: any) => r || []),
  getHourlyModelStats: (date: string, providerID: number, model: string) => StatsService.GetHourlyModelStats(date, providerID, model).then((r: any) => r || []),
  getModelStats: (providerID = 0) => StatsService.GetModelStats(providerID).then((r: any) => r || []),
  getActiveRequests: () => StatsService.GetActiveRequests().then((r: any) => r || []),
  getActiveRequest: (reqId: string) => StatsService.GetActiveRequest(reqId),
  getRecentLogs: (limit = 20, modelName = "") => StatsService.GetRecentLogs(limit, modelName).then((r: any) => r || []),
  getLogDetail: (id: number) => StatsService.GetLogDetail(id),
  // 会话
  getSessions: () => StatsService.GetSessions().then((r: any) => r || []),
  getSessionRequests: (sessionID: number) => StatsService.GetSessionRequests(sessionID).then((r: any) => r || []),
};

// App API
export const AppAPI = {
  getProxyStatus: () => AppService.GetProxyStatus(),
  startProxy: () => AppService.StartProxy(),
  stopProxy: () => AppService.StopProxy(),
  getLogHistory: () => AppService.GetLogHistory().then((r: any) => r || []),
  getNewLogs: () => AppService.GetNewLogs().then((r: any) => r || []),
  getEnvConfig: () => AppService.GetEnvConfig().then((r: any) => r || []),
  saveEnvConfig: (vals: Record<string, string>) => AppService.SaveEnvConfig(vals),
  getVersion: () => AppService.GetVersion(),
  logDebug: (msg: string) => AppService.LogDebug(msg).catch(() => {}),
  logInfo: (msg: string) => AppService.LogInfo(msg).catch(() => {}),
  logWarn: (msg: string) => AppService.LogWarn(msg).catch(() => {}),
  logError: (msg: string) => AppService.LogError(msg).catch(() => {}),
  // 数据库热切换（需要执行 wails generate bindings 或 wails build 后可用）
  testDBConnection: (params: any) => AppService.TestDBConnection(params),
  applyDBConfig: (params: any) => AppService.ApplyDBConfig(params),
  // 开机自启动
  getAutostartStatus: () => AppService.GetAutostartStatus(),
  setAutostart: (enabled: boolean) => AppService.SetAutostart(enabled),
};
