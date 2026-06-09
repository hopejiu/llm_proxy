/** 价格三元组（元/百万 token） */
export interface Prices {
  ip: number; // input_price
  op: number; // output_price
  cp: number; // cache_price
}

export interface TokenUsage {
  total_input_tokens: number;
  total_output_tokens: number;
  total_cached_tokens: number;
}

/** 花费分项 */
export interface CostBreakdown {
  inputCost: number;
  outputCost: number;
  cacheCost: number;
  totalCost: number;
}

/** 花费格式化为 ¥xx.xx */
export function fmtYuan(v: number): string {
  if (v <= 0) return "-";
  if (v >= 10000) return `¥${(v / 10000).toFixed(2)}万`;
  if (v >= 1) return `¥${v.toFixed(2)}`;
  if (v >= 0.01) return `¥${v.toFixed(4)}`;
  return `¥${v.toFixed(6)}`;
}

/** 计算有效输入（扣除缓存命中） */
function paidInput(ms: TokenUsage): number {
  return Math.max(0, ms.total_input_tokens - ms.total_cached_tokens);
}

/** 计算总花费 */
export function computeModelCost(ms: TokenUsage, prices?: Prices): number {
  if (!prices) return 0;
  return (paidInput(ms) / 1_000_000) * prices.ip
    + (ms.total_output_tokens / 1_000_000) * prices.op
    + (ms.total_cached_tokens / 1_000_000) * prices.cp;
}

/** 计算花费分项（输入费/输出费/缓存费/总费） */
export function computeCostBreakdown(ms: TokenUsage, prices?: Prices): CostBreakdown {
  if (!prices) return { inputCost: 0, outputCost: 0, cacheCost: 0, totalCost: 0 };
  const pi = paidInput(ms);
  const inputCost = (pi / 1_000_000) * prices.ip;
  const outputCost = (ms.total_output_tokens / 1_000_000) * prices.op;
  const cacheCost = (ms.total_cached_tokens / 1_000_000) * prices.cp;
  return { inputCost, outputCost, cacheCost, totalCost: inputCost + outputCost + cacheCost };
}

/** 从 providers 数组构建定价查询表: { providerID: { modelName: Prices } } */
export function buildPricingMap(providers: any[]): Record<number, Record<string, Prices>> {
  const map: Record<number, Record<string, Prices>> = {};
  for (const p of providers) {
    let models: any[] = [];
    try { models = JSON.parse(p.models || "[]"); } catch { models = []; }
    const pm: Record<string, Prices> = {};
    for (const m of models) {
      const prices: Prices = { ip: m.input_price || 0, op: m.output_price || 0, cp: m.cache_price || 0 };
      pm[m.name] = prices;
      for (const alias of (m.aliases || [])) pm[alias] = prices;
    }
    map[p.id] = pm;
  }
  return map;
}

/** 从 pricingMap 中查找某条日志对应模型的价格（含跨 provider 兜底） */
export function lookupLogPrices(
  pricingMap: Record<number, Record<string, Prices>>,
  providerId: number,
  model: string,
): Prices | undefined {
  let p = pricingMap[providerId]?.[model];
  if (p) return p;
  for (const pid of Object.keys(pricingMap)) {
    p = pricingMap[Number(pid)]?.[model];
    if (p) return p;
  }
  return undefined;
}
