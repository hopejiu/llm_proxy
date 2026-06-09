import { useState, useEffect, useRef, useCallback } from "react";
import ReactEChartsCore from "echarts-for-react/esm/core";
import * as echarts from "echarts/core";
import { BarChart, LineChart } from "echarts/charts";
import { GridComponent, TooltipComponent, LegendComponent } from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import { StatsService } from "../../bindings/github.com/wanglejiu/llm-proxy";
import { useProviders } from "../hooks/useProviders";
import { useLogDetail } from "../hooks/useLogs";

echarts.use([BarChart, LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const COLORS = ["#3b82f6","#8b5cf6","#10b981","#f59e0b","#ef4444","#06b6d4","#ec4899","#f97316"];
const K = "stats_settings";
function ls() { try { const r = localStorage.getItem(K); if (r) return JSON.parse(r); } catch {} return { autoRefresh: false }; }
function ss(s: any) { try { localStorage.setItem(K, JSON.stringify(s)); } catch {} }
function fd(d: Date) { return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,"0")}-${String(d.getDate()).padStart(2,"0")}`; }
function ds(s: string) { return s ? s.slice(0,10) : ""; }

function fill(stats: any[], min = 7) {
  const t = new Date(); t.setHours(0,0,0,0);
  if (!stats?.length) { const r: any[] = []; for (let i = min-1; i >= 0; i--) { const d = new Date(t); d.setDate(d.getDate()-i); r.push({ date:fd(d), total_input_tokens:0, total_output_tokens:0, total_tokens:0, total_cached_tokens:0, request_count:0 }); } return r; }
  const sorted = [...stats].sort((a,b) => new Date(ds(a.date)).getTime() - new Date(ds(b.date)).getTime());
  const st = new Date(sorted[0].date);
  const df = Math.floor((t.getTime()-st.getTime())/86400000)+1;
  const fs = new Date(df < min ? t.getTime()-(min-1)*86400000 : st.getTime());
  const r: any[] = []; const c = new Date(fs);
  while (c <= t) { const dateStr = fd(c); const f = sorted.find(x => ds(x.date) === dateStr); r.push(f || { date:dateStr, total_input_tokens:0, total_output_tokens:0, total_tokens:0, total_cached_tokens:0, request_count:0 }); c.setDate(c.getDate()+1); }
  return r;
}

function Trend({c,p}:{c:number;p:number}) {
  if (p===0&&c===0) return null;
  if (p===0) return <span className="ml-2 text-[10px] text-green-400">↑ 新增</span>;
  const ch = ((c-p)/p)*100; const ab = Math.abs(ch).toFixed(1);
  if (ch>0) return <span className="ml-2 text-[10px] text-green-400">↑ {ab}%</span>;
  if (ch<0) return <span className="ml-2 text-[10px] text-red-400">↓ {ab}%</span>;
  return <span className="ml-2 text-[10px] text-gray-400">— 0%</span>;
}
function MR({l,v}:{l:string;v?:string}){return v?<div className="flex"><span className="text-xs text-gray-400 w-20 shrink-0">{l}</span><span className="text-sm text-gray-200">{v}</span></div>:null;}
function BK({l,v}:{l:string;v?:string}){return v?<div><span className="text-xs text-gray-400 block mb-1">{l}</span><pre className="bg-[#0f3460] border border-gray-700/50 rounded p-3 text-xs text-gray-300 overflow-auto max-h-60 whitespace-pre-wrap break-all">{v}</pre></div>:null;}
function pj(s:string){try{return JSON.stringify(JSON.parse(s),null,2)}catch{return s}}

const Card = ({l,n,d,tr}:any)=>(
  <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
    <div className="flex items-center justify-between mb-3"><span className="text-xs font-medium text-gray-400">{l}</span><span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-900/40 text-blue-300">{n}</span></div>
    <div className="flex items-baseline gap-1 mb-1"><span className="text-2xl font-bold text-white">{d?.total_tokens!=null?Number(d.total_tokens).toLocaleString():"-"}</span><span className="text-xs text-gray-400">Tokens</span>{tr}</div>
    <div className="grid grid-cols-4 gap-3 mt-3 text-xs">
      {["total_input_tokens","total_output_tokens","total_cached_tokens","request_count"].map(k=>{
        const lab:any={total_input_tokens:"Input",total_output_tokens:"Output",total_cached_tokens:"Cached",request_count:"请求"};
        return <div key={k}><span className="text-gray-500">{lab[k]||k}</span><p className={`font-semibold ${k==="total_cached_tokens"?"text-green-400":"text-gray-200"}`}>{d?.[k]!=null?Number(d[k]).toLocaleString():"-"}</p></div>;
      })}
    </div>
  </div>
);

export default function StatsPage() {
  const { providers } = useProviders();
  const { detail:ld, fetch:fl, clear:cl } = useLogDetail();
  const [sp,setSp] = useState(0);
  const [data,setData] = useState<any>(null);
  const [daily,setDaily] = useState<any[]>([]);
  const [hourly,setHourly] = useState<any[]>([]);
  const [bd,setBd] = useState<any[]>([]);
  const [loading,setLoading] = useState(true);
  const [refreshing,setRefreshing] = useState(false);
  const [sett,setSett] = useState(ls);
  const [dim,setDim] = useState<"tokens"|"requests">("tokens");
  const [hDate,setHDate] = useState(()=>fd(new Date()));
  const [showDD,setShowDD] = useState(false);
  const [recent,setRecent] = useState<any[]>([]);
  const ddRef = useRef<HTMLDivElement>(null);
  const autoRef = useRef<ReturnType<typeof setInterval>|null>(null);
  const isToday = hDate === fd(new Date());
  const maxHour = isToday ? Math.min(new Date().getHours(),23) : 23;
  const stacked = sp === 0 && providers.length > 0;

  const fetchAll = useCallback(async () => {
    setRefreshing(true);
    try {
      const [s,d,h,r] = await Promise.all([StatsService.GetStats(sp),StatsService.GetDailyStats(sp),StatsService.GetHourlyStatsByDate(hDate,sp),StatsService.GetRecentLogs(20)]);
      setData(s); setDaily(d||[]); setHourly(h||[]); setRecent(r||[]);
    } catch {} finally { setRefreshing(false); setLoading(false); }
  }, [sp, hDate]);

  useEffect(()=>{setLoading(true);fetchAll()},[fetchAll]);
  useEffect(()=>{if(autoRef.current)clearInterval(autoRef.current);if(sett.autoRefresh)autoRef.current=setInterval(fetchAll,10000);return()=>{if(autoRef.current)clearInterval(autoRef.current)}},[sett.autoRefresh,fetchAll]);
  useEffect(()=>{function h(e:MouseEvent){if(ddRef.current&&!ddRef.current.contains(e.target as Node))setShowDD(false)}document.addEventListener("click",h);return()=>document.removeEventListener("click",h)},[]);
  useEffect(()=>{if(!stacked){setBd([]);return}StatsService.GetHourlyStatsByDateWithBreakdown(hDate).then((d:any)=>setBd(d||[])).catch(()=>setBd([]))},[hDate,sp,stacked]);

  const tt = data?.today?.total_tokens||0;
  const yt = (()=>{const y=new Date();y.setDate(y.getDate()-1);const ys=fd(y);return daily.find((s:any)=>ds(s.date)===ys)?.total_tokens||0;})();
  const wt = data?.week?.total_tokens||0;
  const lwt = (()=>{const t=new Date();let tot=0;for(let i=7;i<14;i++){const d=new Date(t);d.setDate(d.getDate()-i);tot+=daily.find((s:any)=>ds(s.date)===fd(d))?.total_tokens||0;}return tot;})();

  const filled = fill(daily, 7);
  const cDates = filled.map((d:any)=>{const dt=new Date(d.date);return`${dt.getMonth()+1}/${dt.getDate()}`});
  const sT = [
    {name:"输入 Token",type:"line" as const,smooth:true,data:filled.map((d:any)=>d.total_input_tokens||0),itemStyle:{color:"#7C3AED"},lineStyle:{color:"#7C3AED",width:2}},
    {name:"输出 Token",type:"line" as const,smooth:true,data:filled.map((d:any)=>d.total_output_tokens||0),itemStyle:{color:"#A78BFA"},lineStyle:{color:"#A78BFA",width:2}},
    {name:"Cached Token",type:"line" as const,smooth:true,data:filled.map((d:any)=>d.total_cached_tokens||0),itemStyle:{color:"#10B981"},lineStyle:{color:"#10B981",width:2}},
  ];
  const trendOpt = {
    tooltip:{trigger:"axis" as const,axisPointer:{type:"cross" as const}},
    legend:{data:dim==="tokens"?["输入 Token","输出 Token","Cached Token"]:["请求数"],textStyle:{color:"#9ca3af",fontSize:10},bottom:0},
    grid:{left:"3%",right:"4%",bottom:"14%",top:"8%",containLabel:true},
    xAxis:{type:"category" as const,boundaryGap:false,data:cDates,axisLabel:{color:"#9ca3af",fontSize:10},axisLine:{lineStyle:{color:"#374151"}}},
    yAxis:{type:"value" as const,min:0,axisLabel:{color:"#9ca3af",fontSize:10,formatter:(v:number)=>v>=10000?(v/10000).toFixed(0)+"万":String(v)},splitLine:{lineStyle:{color:"#1f2937"}}},
    series:dim==="tokens"?sT:[{name:"请求数",type:"bar" as const,data:filled.map((d:any)=>d.request_count||0),itemStyle:{color:{type:"linear" as const,x:0,y:0,x2:0,y2:1,colorStops:[{offset:0,color:"#7C3AED"},{offset:1,color:"#A78BFA"}]},borderRadius:[4,4,0,0]as[number,number,number,number]}}],
  };

  const hOpt = (() => {
    if (stacked) return null;
    const hl:string[]=[],td:number[]=[],rd:number[]=[];
    for (let i=0;i<=maxHour;i++) { hl.push(`${i}:00`); const f=hourly.find((s:any)=>s.hour===i); td.push(f?f.total_tokens:0); rd.push(f?f.request_count:0); }
    return { tooltip:{trigger:"axis" as const,axisPointer:{type:"shadow" as const}},legend:{data:["Token 消耗","请求次数"],textStyle:{color:"#9ca3af",fontSize:10},bottom:0},grid:{left:"3%",right:"4%",bottom:"14%",top:"8%",containLabel:true},xAxis:{type:"category" as const,data:hl,axisLabel:{color:"#9ca3af",fontSize:10},axisLine:{lineStyle:{color:"#374151"}}},yAxis:[{type:"value" as const,name:"Token",position:"left" as const,axisLabel:{color:"#9ca3af",fontSize:10,formatter:(v:number)=>v>=10000?(v/10000).toFixed(0)+"万":String(v)},splitLine:{lineStyle:{color:"#1f2937"}}},{type:"value" as const,name:"请求次数",position:"right" as const,axisLabel:{color:"#9ca3af",fontSize:10,formatter:(v:number)=>v>=1000?(v/1000).toFixed(0)+"k":String(v)},splitLine:{show:false}}],series:[{name:"Token 消耗",type:"bar" as const,yAxisIndex:0,data:td,itemStyle:{color:{type:"linear" as const,x:0,y:0,x2:0,y2:1,colorStops:[{offset:0,color:"#7C3AED"},{offset:1,color:"#A78BFA"}]},borderRadius:[4,4,0,0]as[number,number,number,number]}},{name:"请求次数",type:"line" as const,yAxisIndex:1,data:rd,smooth:true,itemStyle:{color:"#F59E0B"},lineStyle:{color:"#F59E0B",width:2}}]};
  })();

  const sOpt = (() => {
    if (!stacked || !bd.length) return null;
    const hl:string[]=[];for(let i=0;i<=maxHour;i++)hl.push(`${i}:00`);
    const hm:Record<number,Record<number,number>>={};for(const i of bd){if(!hm[i.hour])hm[i.hour]={};hm[i.hour][i.provider_id]=i.total_tokens}
    const pIds=[...new Set(bd.map((d:any)=>d.provider_id))];
    return{tooltip:{trigger:"axis" as const,axisPointer:{type:"shadow" as const}},legend:{data:[...new Set(bd.map((d:any)=>d.provider_name))],textStyle:{color:"#9ca3af",fontSize:10},bottom:0},grid:{left:"3%",right:"4%",bottom:"14%",top:"8%",containLabel:true},xAxis:{type:"category" as const,data:hl,axisLabel:{color:"#9ca3af",fontSize:10},axisLine:{lineStyle:{color:"#374151"}}},yAxis:{type:"value" as const,name:"Token",axisLabel:{color:"#9ca3af",fontSize:10,formatter:(v:number)=>v>=10000?(v/10000).toFixed(0)+"万":String(v)},splitLine:{lineStyle:{color:"#1f2937"}}},series:pIds.map((pid:number,i:number)=>{const name=bd.find((d:any)=>d.provider_id===pid)?.provider_name||`P${pid}`;return{name,type:"bar" as const,stack:"total" as const,data:hl.map((_,hi)=>hm[hi]?.[pid]||0),itemStyle:{color:COLORS[i%COLORS.length]}}})};
  })();

  const ddItems = (() => {
    const t=new Date();t.setHours(0,0,0,0);const ts=fd(t);
    const dm:Record<string,number>={};for(const s of daily)dm[ds(s.date)]=s.total_tokens||0;
    const items:any[]=[];for(let i=0;i<30;i++){const d=new Date(t);d.setDate(d.getDate()-i);const dd=fd(d);const wd=["日","一","二","三","四","五","六"][d.getDay()];items.push({ds:dd,tokens:dm[dd]||0,wd,isT:dd===ts})}
    return{items,maxT:Math.max(...items.map((it:any)=>it.tokens),1)};
  })();

  return (<div className="p-6 space-y-6">
    <div className="flex items-center justify-between">
      <h1 className="text-xl font-bold text-white">统计仪表盘</h1>
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 text-xs text-gray-400">
          <span>自动刷新</span>
          <label className="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" checked={sett.autoRefresh} onChange={()=>setSett((p:any)=>{const n={autoRefresh:!p.autoRefresh};ss(n);return n})} className="sr-only peer" />
            <div className="w-8 h-4 bg-gray-600 rounded-full peer peer-checked:bg-blue-600 peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-3 after:w-3 after:transition-all" />
          </label>
        </div>
        <button onClick={fetchAll} disabled={refreshing} className="p-1.5 rounded hover:bg-gray-700/30 text-gray-400 hover:text-gray-200 transition-colors disabled:opacity-50">
          <svg className={`w-4 h-4 ${refreshing?"animate-spin":""}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
        </button>
        <select value={sp} onChange={e=>setSp(Number(e.target.value))} className="bg-[#0f3460] border border-gray-600 rounded px-3 py-1.5 text-sm text-white focus:outline-none focus:border-blue-500">
          <option value={0}>全部 Provider</option>
          {providers.map((p:any)=><option key={p.id} value={p.id}>{p.name}</option>)}
        </select>
      </div>
    </div>

    {loading ? <div className="text-center py-20 text-gray-400">加载中...</div> : (
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card l="今日用量" n="Today" d={data?.today} tr={<Trend c={tt} p={yt}/>} />
        <Card l="本周用量" n="Week" d={data?.week} tr={<Trend c={wt} p={lwt}/>} />
        <Card l="总计用量" n="Total" d={data?.total} tr={null} />
      </div>
    )}

    <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-sm font-semibold text-gray-200">分时用量</h2>
        <div className="flex items-center gap-2">
          <button onClick={()=>{const d=new Date(hDate+"T00:00:00");d.setDate(d.getDate()-1);setHDate(fd(d))}} className="p-1 rounded hover:bg-gray-700/30 text-gray-400 hover:text-gray-200">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7"/></svg></button>
          <div className="relative" ref={ddRef}>
            <button onClick={()=>setShowDD(v=>!v)} className="flex items-center gap-1 px-2 py-1 text-xs text-gray-300 bg-[#0f3460] rounded hover:bg-gray-700/30">
              <span>{isToday?"今天":`${new Date(hDate+"T00:00:00").getMonth()+1}/${new Date(hDate+"T00:00:00").getDate()}`}</span>
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7"/></svg></button>
            {showDD&&<div className="absolute right-0 top-full mt-1 w-64 bg-[#16213e] border border-gray-600 rounded-lg shadow-xl z-50 max-h-80 overflow-y-auto">
              <div className="sticky top-0 bg-[#16213e] px-3 py-1.5 border-b border-gray-700/50 flex items-center gap-2 text-[10px] text-gray-400"><span className="flex-1">日期</span><span>消耗量</span></div>
              {ddItems.items.map((item:any)=>(<div key={item.ds} onClick={()=>{setHDate(item.ds);setShowDD(false)}} className={`flex items-center gap-2 px-3 py-1.5 text-xs cursor-pointer transition-colors hover:bg-gray-700/20 ${item.ds===hDate?"bg-blue-900/30":""}`}>
                <span className="w-24 shrink-0 text-gray-300">{item.isT?"今天":`${item.ds.slice(5)} 周${item.wd}`}</span>
                <div className="flex-1 h-2 bg-gray-700 rounded-full overflow-hidden"><div className="h-full bg-purple-500 rounded-full" style={{width:`${(item.tokens/ddItems.maxT)*100}%`}}/></div>
                <span className="w-16 text-right text-gray-400 shrink-0">{item.tokens>0?item.tokens>=10000?(item.tokens/10000).toFixed(1)+"万":item.tokens.toLocaleString():"-"}</span>
              </div>))}
            </div>}
          </div>
          <button onClick={()=>{const d=new Date(hDate+"T00:00:00");d.setDate(d.getDate()+1);setHDate(fd(d))}} className="p-1 rounded hover:bg-gray-700/30 text-gray-400 hover:text-gray-200">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7"/></svg></button>
          {!isToday&&<button onClick={()=>setHDate(fd(new Date()))} className="px-2 py-1 text-[10px] bg-[#0f3460] rounded text-gray-300 hover:bg-gray-700/30">今天</button>}
        </div>
      </div>
      <div style={{minHeight:300}}>{stacked&&sOpt?<ReactEChartsCore echarts={echarts} option={sOpt} style={{height:300}} notMerge />:hOpt?<ReactEChartsCore echarts={echarts} option={hOpt} style={{height:300}} notMerge />:<div className="h-[300px] flex items-center justify-center text-gray-500 text-sm">暂无分时数据</div>}</div>
    </div>

    <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-sm font-semibold text-gray-200">近 30 天趋势</h2>
        <div className="flex items-center gap-1 bg-[#0f3460] rounded text-xs">
          <button onClick={()=>setDim("tokens")} className={`px-2.5 py-1 rounded transition-colors ${dim==="tokens"?"bg-blue-600/40 text-blue-300":"text-gray-400 hover:text-gray-200"}`}>Token 用量</button>
          <button onClick={()=>setDim("requests")} className={`px-2.5 py-1 rounded transition-colors ${dim==="requests"?"bg-blue-600/40 text-blue-300":"text-gray-400 hover:text-gray-200"}`}>请求数</button>
        </div>
      </div>
      <ReactEChartsCore echarts={echarts} option={trendOpt} style={{height:300}} notMerge />
    </div>

    <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
      <h2 className="text-sm font-semibold text-gray-200 mb-3">每日明细</h2>
      <div className="overflow-x-auto"><table className="w-full text-xs">
        <thead><tr className="border-b border-gray-700/50 text-gray-400">
          <th className="text-left py-2 px-2">日期</th><th className="text-right py-2 px-2">请求数</th>
          <th className="text-right py-2 px-2 hidden sm:table-cell">Input</th><th className="text-right py-2 px-2 hidden sm:table-cell">Output</th>
          <th className="text-right py-2 px-2 hidden sm:table-cell text-green-400">Cached</th><th className="text-right py-2 px-2 font-semibold text-purple-400">Total</th>
        </tr></thead>
        <tbody>{daily.length===0?<tr><td colSpan={6} className="text-center py-12 text-gray-500">暂无数据</td></tr>
          :[...daily].reverse().map((s:any,i:number)=>(<tr key={i} className="border-b border-gray-800/30 hover:bg-gray-700/10">
            <td className="py-2 px-2 text-gray-300">{ds(s.date)}</td>
            <td className="py-2 px-2 text-right text-gray-300">{s.request_count?.toLocaleString()||"-"}</td>
            <td className="py-2 px-2 text-right text-gray-300 hidden sm:table-cell">{s.total_input_tokens?.toLocaleString()||"-"}</td>
            <td className="py-2 px-2 text-right text-gray-300 hidden sm:table-cell">{s.total_output_tokens?.toLocaleString()||"-"}</td>
            <td className="py-2 px-2 text-right text-green-400 hidden sm:table-cell">{s.total_cached_tokens?.toLocaleString()||"-"}</td>
            <td className="py-2 px-2 text-right font-semibold text-purple-400">{s.total_tokens?.toLocaleString()||"-"}</td>
          </tr>))}</tbody>
      </table></div>
    </div>

    <div className="bg-[#16213e] border border-gray-700/50 rounded-lg p-4">
      <h2 className="text-sm font-semibold text-gray-200 mb-3">最近请求</h2>
      <div className="overflow-x-auto"><table className="w-full text-xs">
        <thead><tr className="border-b border-gray-700/50 text-gray-400">
          <th className="text-left py-2 px-2">时间</th><th className="text-left py-2 px-2">Provider</th><th className="text-left py-2 px-2">模型</th>
          <th className="text-right py-2 px-2 hidden sm:table-cell">Input</th><th className="text-right py-2 px-2 hidden sm:table-cell">Output</th><th className="text-right py-2 px-2 hidden sm:table-cell text-green-400">Cached</th>
          <th className="text-right py-2 px-2">Total</th><th className="text-right py-2 px-2 hidden sm:table-cell">耗时</th><th className="text-right py-2 px-2 hidden sm:table-cell">Token/s</th><th className="text-center py-2 px-2">状态</th><th className="text-center py-2 px-2">操作</th>
        </tr></thead>
        <tbody>{recent.length===0?<tr><td colSpan={11} className="text-center py-12 text-gray-500">暂无请求记录</td></tr>
          :recent.map((lg:any)=>{const tps=lg.duration>0?(lg.output_tokens*1000/lg.duration).toFixed(1):"-";return(
            <tr key={lg.id} className="border-b border-gray-800/30 hover:bg-gray-700/10">
              <td className="py-2 px-2 text-gray-400 whitespace-nowrap">{lg.created_at}</td>
              <td className="py-2 px-2 text-gray-300 max-w-[80px] truncate">{lg.provider_name||"-"}</td>
              <td className="py-2 px-2 text-gray-300 max-w-[100px] truncate">{lg.model}</td>
              <td className="py-2 px-2 text-right text-gray-300 hidden sm:table-cell">{lg.input_tokens?.toLocaleString()||"-"}</td>
              <td className="py-2 px-2 text-right text-gray-300 hidden sm:table-cell">{lg.output_tokens?.toLocaleString()||"-"}</td>
              <td className="py-2 px-2 text-right text-green-400 hidden sm:table-cell">{lg.cached_tokens?.toLocaleString()||"-"}</td>
              <td className="py-2 px-2 text-right font-semibold text-purple-400">{lg.total_tokens?.toLocaleString()||"-"}</td>
              <td className="py-2 px-2 text-right text-gray-400 hidden sm:table-cell">{lg.duration>0?`${(lg.duration/1000).toFixed(1)}s`:"-"}</td>
              <td className="py-2 px-2 text-right text-blue-400 font-medium hidden sm:table-cell">{tps}</td>
              <td className="py-2 px-2 text-center"><span className={`text-[10px] px-1.5 py-0.5 rounded ${lg.status==="success"?"bg-green-900/40 text-green-300":"bg-red-900/40 text-red-300"}`}>{lg.status==="success"?"成功":"失败"}</span></td>
              <td className="py-2 px-2 text-center"><button onClick={()=>fl(lg.id)} className="text-purple-400 hover:text-purple-300 text-xs">查看</button></td>
            </tr>);})}</tbody>
      </table></div>
    </div>

    {ld&&<div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50" onClick={cl}>
      <div className="bg-[#16213e] border border-gray-600 rounded-lg w-full max-w-3xl mx-4 max-h-[85vh] flex flex-col" onClick={e=>e.stopPropagation()}>
        <div className="flex items-center justify-between p-4 border-b border-gray-700/50"><h2 className="text-base font-semibold text-white">日志详情</h2><button onClick={cl} className="text-gray-400 hover:text-white text-sm">关闭</button></div>
        <div className="flex-1 overflow-auto p-4 space-y-3">
          <MR l="Provider" v={ld.provider_name}/><MR l="模型" v={ld.model}/>
          <MR l="Token" v={`输入 ${ld.input_tokens} / 输出 ${ld.output_tokens} / 总计 ${ld.total_tokens}`}/>
          {ld.cached_tokens>0&&<MR l="缓存 Token" v={String(ld.cached_tokens)}/>}
          <MR l="状态" v={ld.status}/><MR l="错误信息" v={ld.error_message}/><MR l="耗时" v={`${(ld.duration/1000).toFixed(1)}s`}/>
          <BK l="请求体" v={pj(ld.request_body)}/><BK l="响应体" v={pj(ld.response_body)}/>
          {ld.thinking_content&&<BK l="思考内容" v={ld.thinking_content}/>}
        </div>
      </div>
    </div>}
  </div>);
}
