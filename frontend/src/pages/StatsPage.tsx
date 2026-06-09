import { useState, useEffect, useRef, useCallback } from "react";
import ReactEChartsCore from "echarts-for-react/esm/core";
import * as echarts from "echarts/core";
import { BarChart, LineChart } from "echarts/charts";
import { GridComponent, TooltipComponent, LegendComponent } from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import { StatsAPI } from "../services";
import { useProviders } from "../hooks/useProviders";
import { useLocalStorage } from "../hooks/useLocalStorage";
import Skeleton from "../components/Skeleton";
import RecentRequestsTable from "../components/RecentRequestsTable";

echarts.use([BarChart, LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const PURPLE = ["#7C3AED","#A78BFA","#C4B5FD","#8B5CF6","#6D28D9","#5B21B6","#10B981","#F59E0B"];
function fd(d: Date) { return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,"0")}-${String(d.getDate()).padStart(2,"0")}`; }
function ds(s: string) { return s ? s.slice(0,10) : ""; }
function cacheRate(cached?: number|null, input?: number|null): string {
  if (!cached || !input || input === 0) return "-";
  return (cached / input * 100).toFixed(1) + "%";
}

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
  if (p===0) return <span className="ml-2 text-[10px] text-emerald-600 font-medium">↑ 新增</span>;
  const ch = ((c-p)/p)*100; const ab = Math.abs(ch).toFixed(1);
  if (ch>0) return <span className="ml-2 text-[10px] text-emerald-600 font-medium">↑ {ab}%</span>;
  if (ch<0) return <span className="ml-2 text-[10px] text-red-500 font-medium">↓ {ab}%</span>;
  return <span className="ml-2 text-[10px] text-[#9C94B0]">— 0%</span>;
}

function Card({l,n,d,tr}:any){
  const items = [
    {k:"total_input_tokens", lab:"Input"},
    {k:"total_output_tokens", lab:"Output"},
    {k:"total_cached_tokens", lab:"Cached"},
    {k:"request_count", lab:"请求"},
  ];
  return (
    <div className="stat-card">
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs font-semibold text-[#6B6580] uppercase tracking-wider">{l}</span>
        <span className="badge-purple">{n}</span>
      </div>
      <div className="flex items-baseline gap-1.5 mb-1">
        <span className="text-2xl font-bold text-[#1E1B2E] font-heading">
          {d?.total_tokens!=null ? Number(d.total_tokens).toLocaleString() : "-"}
        </span>
        <span className="text-xs text-[#9C94B0]">Tokens{tr}</span>
      </div>
      <div className="grid grid-cols-4 gap-x-1.5 gap-y-1 mt-4 pt-3 border-t border-[#F0EBF5]">
        {items.map(({k,lab})=>(
          <div key={k} className="min-w-0">
            <span className="text-[10px] text-[#9C94B0]">{lab}</span>
            <p className={`text-sm font-semibold mt-0.5 truncate ${k==="total_cached_tokens"?"text-emerald-600":"text-[#1E1B2E]"}`}
               title={d?.[k]!=null ? Number(d[k]).toLocaleString() : "-"}>
              {d?.[k]!=null ? Number(d[k]).toLocaleString() : "-"}
            </p>
            {k === "total_cached_tokens" && (
              <span className="text-[9px] text-emerald-500 font-medium">{cacheRate(d?.total_cached_tokens, d?.total_input_tokens)}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function StatsPage() {
  const { providers } = useProviders();
  const [sp,setSp] = useState(0);
  const [data,setData] = useState<any>(null);
  const [daily,setDaily] = useState<any[]>([]);
  const [hourly,setHourly] = useState<any[]>([]);
  const [bd,setBd] = useState<any[]>([]);
  const [loading,setLoading] = useState(true);
  const [refreshing,setRefreshing] = useState(false);
  const [sett,setSett] = useLocalStorage("stats_settings", { autoRefresh: false });
  const [dim,setDim] = useState<"tokens"|"requests">("tokens");
  const [hDate,setHDate] = useState(()=>fd(new Date()));
  const [showDD,setShowDD] = useState(false);
  const [recent,setRecent] = useState<any[]>([]);
  const [updateTime,setUpdateTime] = useState("");
  const ddRef = useRef<HTMLDivElement>(null);
  const autoRef = useRef<ReturnType<typeof setInterval>|null>(null);
  const isToday = hDate === fd(new Date());
  const maxHour = isToday ? Math.min(new Date().getHours(),23) : 23;
  const stacked = sp === 0 && providers.length > 0;

  const fetchAll = useCallback(async () => {
    setRefreshing(true);
    try {
      const [s,d,h,r] = await Promise.all([StatsAPI.getStats(sp),StatsAPI.getDailyStats(sp),StatsAPI.getHourlyStatsByDate(hDate,sp),StatsAPI.getRecentLogs(20)]);
      setData(s); setDaily(d||[]); setHourly(h||[]); setRecent(r||[]);
      setUpdateTime(new Date().toLocaleTimeString());
    } catch {} finally { setRefreshing(false); setLoading(false); }
  }, [sp, hDate]);

  useEffect(()=>{setLoading(true);fetchAll()},[fetchAll]);
  useEffect(()=>{if(autoRef.current)clearInterval(autoRef.current);if(sett.autoRefresh)autoRef.current=setInterval(fetchAll,10000);return()=>{if(autoRef.current)clearInterval(autoRef.current)}},[sett.autoRefresh,fetchAll]);
  useEffect(()=>{function h(e:MouseEvent){if(ddRef.current&&!ddRef.current.contains(e.target as Node))setShowDD(false)}document.addEventListener("click",h);return()=>document.removeEventListener("click",h)},[]);
  useEffect(()=>{if(!stacked){setBd([]);return}StatsAPI.getHourlyStatsByDateWithBreakdown(hDate).then((d:any)=>setBd(d||[])).catch(()=>setBd([]))},[hDate,sp,stacked]);

  const tt = data?.today?.total_tokens||0;
  const yt = (()=>{const y=new Date();y.setDate(y.getDate()-1);const ys=fd(y);return daily.find((s:any)=>ds(s.date)===ys)?.total_tokens||0;})();
  const wt = data?.week?.total_tokens||0;
  const lwt = (()=>{const t=new Date();let tot=0;for(let i=7;i<14;i++){const d=new Date(t);d.setDate(d.getDate()-i);tot+=daily.find((s:any)=>ds(s.date)===fd(d))?.total_tokens||0;}return tot;})();

  const filled = fill(daily, 7);
  const cDates = filled.map((d:any)=>{const dt=new Date(d.date);return`${dt.getMonth()+1}/${dt.getDate()}`});
  const sT = [
    {name:"输入 Token",type:"line" as const,smooth:true,data:filled.map((d:any)=>d.total_input_tokens||0),itemStyle:{color:"#7C3AED"},lineStyle:{color:"#7C3AED",width:2},areaStyle:{color:{type:"linear" as const,x:0,y:0,x2:0,y2:1,colorStops:[{offset:0,color:"rgba(124,58,237,0.15)"},{offset:1,color:"rgba(124,58,237,0)"}]}}},
    {name:"输出 Token",type:"line" as const,smooth:true,data:filled.map((d:any)=>d.total_output_tokens||0),itemStyle:{color:"#A78BFA"},lineStyle:{color:"#A78BFA",width:2},areaStyle:{color:{type:"linear" as const,x:0,y:0,x2:0,y2:1,colorStops:[{offset:0,color:"rgba(167,139,250,0.15)"},{offset:1,color:"rgba(167,139,250,0)"}]}}},
    {name:"Cached Token",type:"line" as const,smooth:true,data:filled.map((d:any)=>d.total_cached_tokens||0),itemStyle:{color:"#10B981"},lineStyle:{color:"#10B981",width:2},areaStyle:{color:{type:"linear" as const,x:0,y:0,x2:0,y2:1,colorStops:[{offset:0,color:"rgba(16,185,129,0.15)"},{offset:1,color:"rgba(16,185,129,0)"}]}}},
  ];
  const trendOpt = {
    tooltip:{trigger:"axis" as const,axisPointer:{type:"cross" as const,crossStyle:{color:"#9C94B0"}}},
    legend:{data:dim==="tokens"?["输入 Token","输出 Token","Cached Token"]:["请求数"],textStyle:{color:"#6B6580",fontSize:11},bottom:0},
    grid:{left:"3%",right:"4%",bottom:"14%",top:"8%",containLabel:true},
    xAxis:{type:"category" as const,boundaryGap:false,data:cDates,axisLabel:{color:"#9C94B0",fontSize:10},axisLine:{lineStyle:{color:"#EDE9FE"}},axisTick:{lineStyle:{color:"#EDE9FE"}}},
    yAxis:{type:"value" as const,min:0,axisLabel:{color:"#9C94B0",fontSize:10,formatter:(v:number)=>v>=10000?(v/10000).toFixed(0)+"万":String(v)},splitLine:{lineStyle:{color:"#F0EBF5"}}},
    series:dim==="tokens"?sT:[{name:"请求数",type:"bar" as const,data:filled.map((d:any)=>d.request_count||0),itemStyle:{color:{type:"linear" as const,x:0,y:0,x2:0,y2:1,colorStops:[{offset:0,color:"#7C3AED"},{offset:1,color:"#A78BFA"}]},borderRadius:[4,4,0,0]as[number,number,number,number]}}],
  };

  const hOpt = (() => {
    if (stacked) return null;
    const hl:string[]=[],td:number[]=[],rd:number[]=[];
    for (let i=0;i<=maxHour;i++) { hl.push(`${i}:00`); const f=hourly.find((s:any)=>s.hour===i); td.push(f?f.total_tokens:0); rd.push(f?f.request_count:0); }
    return { tooltip:{trigger:"axis" as const,axisPointer:{type:"shadow" as const}},legend:{data:["Token 消耗","请求次数"],textStyle:{color:"#6B6580",fontSize:11},bottom:0},grid:{left:"3%",right:"4%",bottom:"14%",top:"8%",containLabel:true},xAxis:{type:"category" as const,data:hl,axisLabel:{color:"#9C94B0",fontSize:10},axisLine:{lineStyle:{color:"#EDE9FE"}},axisTick:{lineStyle:{color:"#EDE9FE"}}},yAxis:[{type:"value" as const,name:"Token",position:"left" as const,axisLabel:{color:"#9C94B0",fontSize:10,formatter:(v:number)=>v>=10000?(v/10000).toFixed(0)+"万":String(v)},splitLine:{lineStyle:{color:"#F0EBF5"}},nameTextStyle:{color:"#6B6580",fontSize:10}},{type:"value" as const,name:"请求次数",position:"right" as const,axisLabel:{color:"#9C94B0",fontSize:10,formatter:(v:number)=>v>=1000?(v/1000).toFixed(0)+"k":String(v)},splitLine:{show:false},nameTextStyle:{color:"#6B6580",fontSize:10}}],series:[{name:"Token 消耗",type:"bar" as const,yAxisIndex:0,data:td,itemStyle:{color:{type:"linear" as const,x:0,y:0,x2:0,y2:1,colorStops:[{offset:0,color:"#7C3AED"},{offset:1,color:"#A78BFA"}]},borderRadius:[4,4,0,0]as[number,number,number,number]}},{name:"请求次数",type:"line" as const,yAxisIndex:1,data:rd,smooth:true,itemStyle:{color:"#F59E0B"},lineStyle:{color:"#F59E0B",width:2}}]};
  })();

  const sOpt = (() => {
    if (!stacked || !bd.length) return null;
    const hl:string[]=[];for(let i=0;i<=maxHour;i++)hl.push(`${i}:00`);
    const hm:Record<number,Record<number,number>>={};for(const i of bd){if(!hm[i.hour])hm[i.hour]={};hm[i.hour][i.provider_id]=i.total_tokens}
    const pIds=[...new Set(bd.map((d:any)=>d.provider_id))] as number[];
    return{tooltip:{trigger:"axis" as const,axisPointer:{type:"shadow" as const}},legend:{data:[...new Set(bd.map((d:any)=>d.provider_name))],textStyle:{color:"#6B6580",fontSize:11},bottom:0},grid:{left:"3%",right:"4%",bottom:"14%",top:"8%",containLabel:true},xAxis:{type:"category" as const,data:hl,axisLabel:{color:"#9C94B0",fontSize:10},axisLine:{lineStyle:{color:"#EDE9FE"}}},yAxis:{type:"value" as const,name:"Token",axisLabel:{color:"#9C94B0",fontSize:10,formatter:(v:number)=>v>=10000?(v/10000).toFixed(0)+"万":String(v)},splitLine:{lineStyle:{color:"#F0EBF5"}},nameTextStyle:{color:"#6B6580",fontSize:10}},series:pIds.map((pid:number,i:number)=>{const name=bd.find((d:any)=>d.provider_id===pid)?.provider_name||`P${pid}`;return{name,type:"bar" as const,stack:"total" as const,data:hl.map((_,hi)=>hm[hi]?.[pid]||0),itemStyle:{color:PURPLE[i%PURPLE.length]}}})};
  })();

  const ddItems = (() => {
    const t=new Date();t.setHours(0,0,0,0);const ts=fd(t);
    const dm:Record<string,number>={};for(const s of daily)dm[ds(s.date)]=s.total_tokens||0;
    const items:any[]=[];for(let i=0;i<30;i++){const d=new Date(t);d.setDate(d.getDate()-i);const dd=fd(d);const wd=["日","一","二","三","四","五","六"][d.getDay()];items.push({ds:dd,tokens:dm[dd]||0,wd,isT:dd===ts})}
    return{items,maxT:Math.max(...items.map((it:any)=>it.tokens),1)};
  })();

  return (<div className="page-container">
    <div className="page-header">
      <h1 className="page-title">统计仪表盘</h1>
      <div className="flex items-center gap-3">
        <span className="text-xs text-[#9C94B0]">{updateTime ? `更新于 ${updateTime}` : ""}</span>
        <div className="flex items-center gap-2 text-xs text-[#6B6580]">
          <span>自动刷新</span>
          <label className="toggle">
            <input type="checkbox" checked={sett.autoRefresh} onChange={()=>setSett({autoRefresh:!sett.autoRefresh})} className="sr-only peer" />
            <div className="toggle-track peer-checked:bg-brand-600" />
            <div className="toggle-thumb peer-checked:translate-x-4" />
          </label>
        </div>
        <button onClick={fetchAll} disabled={refreshing} className="btn-ghost p-1.5 disabled:opacity-50" title="刷新">
          <svg className={`w-4 h-4 ${refreshing?"animate-spin":""}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
        </button>
        <select value={sp} onChange={e=>setSp(Number(e.target.value))} className="input-field w-auto text-xs">
          <option value={0}>全部 Provider</option>
          {providers.map((p:any)=><option key={p.id} value={p.id}>{p.name}</option>)}
        </select>
      </div>
    </div>

    {loading ? (
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="stat-card"><Skeleton className="h-4 w-16 mb-3" /><Skeleton className="h-8 w-24 mb-1" /><Skeleton className="h-12 w-full mt-4" /></div>
        <div className="stat-card"><Skeleton className="h-4 w-16 mb-3" /><Skeleton className="h-8 w-24 mb-1" /><Skeleton className="h-12 w-full mt-4" /></div>
        <div className="stat-card"><Skeleton className="h-4 w-16 mb-3" /><Skeleton className="h-8 w-24 mb-1" /><Skeleton className="h-12 w-full mt-4" /></div>
      </div>
    ) : (
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card l="今日用量" n="Today" d={data?.today} tr={<Trend c={tt} p={yt}/>} />
        <Card l="本周用量" n="Week" d={data?.week} tr={<Trend c={wt} p={lwt}/>} />
        <Card l="总计用量" n="Total" d={data?.total} tr={null} />
      </div>
    )}

    <div className="section-card">
      <div className="card-header">
        <h2 className="card-title">分时用量</h2>
        <div className="flex items-center gap-2">
          <button onClick={()=>{const d=new Date(hDate+"T00:00:00");d.setDate(d.getDate()-1);setHDate(fd(d))}} className="btn-ghost p-1">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7"/></svg>
          </button>
          <div className="relative" ref={ddRef}>
            <button onClick={()=>setShowDD(v=>!v)} className="flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-[#6B6580] bg-brand-50 border border-brand-200 rounded-lg hover:bg-brand-100 transition-colors">
              <span>{isToday?"今天":`${new Date(hDate+"T00:00:00").getMonth()+1}/${new Date(hDate+"T00:00:00").getDate()}`}</span>
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7"/></svg>
            </button>
            {showDD&&<div className="absolute right-0 top-full mt-1 w-72 bg-white border border-[#EDE9FE] rounded-xl shadow-xl z-50 max-h-80 overflow-y-auto">
              <div className="sticky top-0 bg-white px-4 py-2 border-b border-[#F0EBF5] flex items-center gap-2 text-[10px] font-semibold text-[#6B6580] uppercase tracking-wider"><span className="flex-1">日期</span><span>消耗量</span></div>
              {ddItems.items.map((item:any)=>(
                <div key={item.ds} onClick={()=>{setHDate(item.ds);setShowDD(false)}} className={`flex items-center gap-2 px-4 py-2 text-xs cursor-pointer transition-colors hover:bg-brand-50 ${item.ds===hDate?"bg-brand-50":""}`}>
                  <span className="w-24 shrink-0 font-medium text-[#1E1B2E]">{item.isT?"今天":`${item.ds.slice(5)} 周${item.wd}`}</span>
                  <div className="flex-1 h-2 bg-[#F0EBF5] rounded-full overflow-hidden"><div className="h-full bg-brand-600 rounded-full" style={{width:`${(item.tokens/ddItems.maxT)*100}%`}}/></div>
                  <span className="w-20 text-right text-[#6B6580] shrink-0 font-mono text-[11px]">{item.tokens>0?item.tokens>=10000?(item.tokens/10000).toFixed(1)+"万":item.tokens.toLocaleString():"-"}</span>
                </div>
              ))}
            </div>}
          </div>
          <button onClick={()=>{const d=new Date(hDate+"T00:00:00");d.setDate(d.getDate()+1);setHDate(fd(d))}} className="btn-ghost p-1">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7"/></svg>
          </button>
          {!isToday&&<button onClick={()=>setHDate(fd(new Date()))} className="px-2.5 py-1 text-[10px] font-medium bg-brand-50 text-brand-700 rounded-lg hover:bg-brand-100 transition-colors">今天</button>}
        </div>
      </div>
      <div className="echarts-light" style={{minHeight:300}}>{stacked&&sOpt?<ReactEChartsCore echarts={echarts} option={sOpt} style={{height:300}} notMerge />:hOpt?<ReactEChartsCore echarts={echarts} option={hOpt} style={{height:300}} notMerge />:<div className="h-[300px] flex items-center justify-center text-[#9C94B0] text-sm">暂无分时数据</div>}</div>
    </div>

    <div className="section-card">
      <div className="card-header">
        <h2 className="card-title">近 30 天趋势</h2>
        <div className="flex items-center gap-1 bg-brand-50 rounded-lg p-0.5">
          <button onClick={()=>setDim("tokens")} className={`px-3 py-1 text-xs font-medium rounded-md transition-all duration-150 ${dim==="tokens"?"bg-white text-brand-700 shadow-sm":"text-[#6B6580] hover:text-brand-600"}`}>Token 用量</button>
          <button onClick={()=>setDim("requests")} className={`px-3 py-1 text-xs font-medium rounded-md transition-all duration-150 ${dim==="requests"?"bg-white text-brand-700 shadow-sm":"text-[#6B6580] hover:text-brand-600"}`}>请求数</button>
        </div>
      </div>
      <div className="echarts-light"><ReactEChartsCore echarts={echarts} option={trendOpt} style={{height:300}} notMerge /></div>
    </div>

    <div className="section-card">
      <div className="card-header">
        <h2 className="card-title">每日明细</h2>
      </div>
      <div className="table-wrap">
        <table className="table-base text-xs">
          <thead><tr className="border-b border-[#F0EBF5]">
            <th className="table-th">日期</th><th className="table-th text-right">请求数</th>
            <th className="table-th text-right hidden sm:table-cell">Input</th><th className="table-th text-right hidden sm:table-cell">Output</th>
            <th className="table-th text-right hidden sm:table-cell text-emerald-600">Cached</th><th className="table-th text-right hidden sm:table-cell text-emerald-600">命中率</th><th className="table-th text-right font-semibold text-brand-700">Total</th>
          </tr></thead>
          <tbody>{daily.length===0?<tr><td colSpan={7} className="text-center py-12 text-[#9C94B0]">暂无数据</td></tr>
            :[...daily].reverse().map((s:any,i:number)=>(<tr key={i} className="table-tr">
            <td className="table-td text-[#6B6580] font-mono text-xs">{ds(s.date)}</td>
            <td className="table-td text-right font-mono text-xs">{s.request_count?.toLocaleString()||"-"}</td>
            <td className="table-td text-right font-mono text-xs hidden sm:table-cell">{s.total_input_tokens?.toLocaleString()||"-"}</td>
            <td className="table-td text-right font-mono text-xs hidden sm:table-cell">{s.total_output_tokens?.toLocaleString()||"-"}</td>
            <td className="table-td text-right font-mono text-xs text-emerald-600 hidden sm:table-cell">{s.total_cached_tokens?.toLocaleString()||"-"}</td>
            <td className="table-td text-right font-mono text-xs text-emerald-500 hidden sm:table-cell">{cacheRate(s.total_cached_tokens, s.total_input_tokens)}</td>
            <td className="table-td text-right font-mono text-xs font-bold text-brand-700">{s.total_tokens?.toLocaleString()||"-"}</td>
          </tr>))}</tbody>
        </table>
      </div>
    </div>

    <div className="section-card">
      <div className="card-header">
        <h2 className="card-title">最近请求</h2>
      </div>
      <RecentRequestsTable logs={recent} showTps emptyText="暂无请求记录" />
    </div>
  </div>);
}
