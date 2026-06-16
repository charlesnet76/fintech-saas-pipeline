import { useState, useEffect, useRef } from 'react';
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  AreaChart, Area, ReferenceLine
} from 'recharts';

const BASE_URL = 'https://pipeline-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io';

const monthlyRevenue = [
  { m: 'JAN', total_revenue: 152671.79, total_transactions: 312, avg_transaction: 489.33, unique_customers: 153 },
  { m: 'FEB', total_revenue: 127804.21, total_transactions: 244, avg_transaction: 523.79, unique_customers: 141 },
  { m: 'MAR', total_revenue: 157035.07, total_transactions: 307, avg_transaction: 511.51, unique_customers: 156 },
  { m: 'APR', total_revenue: 144573.63, total_transactions: 271, avg_transaction: 533.48, unique_customers: 153 },
  { m: 'MAY', total_revenue: 117303.60, total_transactions: 289, avg_transaction: 405.89, unique_customers: 147 },
  { m: 'JUN', total_revenue: 122352.97, total_transactions: 257, avg_transaction: 476.08, unique_customers: 139 },
  { m: 'JUL', total_revenue: 129147.45, total_transactions: 296, avg_transaction: 436.31, unique_customers: 151 },
  { m: 'AUG', total_revenue: 143422.08, total_transactions: 287, avg_transaction: 499.73, unique_customers: 155 },
  { m: 'SEP', total_revenue: 107275.04, total_transactions: 272, avg_transaction: 394.39, unique_customers: 149 },
  { m: 'OCT', total_revenue: 134128.34, total_transactions: 285, avg_transaction: 470.63, unique_customers: 157 },
  { m: 'NOV', total_revenue: 105719.53, total_transactions: 248, avg_transaction: 426.29, unique_customers: 146 },
  { m: 'DEC', total_revenue: 156466.16, total_transactions: 305, avg_transaction: 513.00, unique_customers: 168 },
];

const revenueByCategory = [
  { category: 'TRANSFERS',      total_transactions: 350, total_revenue: 880047.85, avg_transaction: 2514.42 },
  { category: 'TRAVEL',         total_transactions: 329, total_revenue: 322604.38, avg_transaction:  980.56 },
  { category: 'SHOPPING',       total_transactions: 333, total_revenue: 130442.78, avg_transaction:  391.72 },
  { category: 'HEALTHCARE',     total_transactions: 341, total_revenue:  91624.64, avg_transaction:  268.69 },
  { category: 'GROCERIES',      total_transactions: 378, total_revenue:  56695.58, avg_transaction:  149.99 },
  { category: 'UTILITIES',      total_transactions: 298, total_revenue:  40951.32, avg_transaction:  137.42 },
  { category: 'ENTERTAINMENT',  total_transactions: 324, total_revenue:  31807.80, avg_transaction:   98.17 },
  { category: 'FOOD+DRINK',     total_transactions: 344, total_revenue:  20719.72, avg_transaction:   60.23 },
  { category: 'TRANSPORT',      total_transactions: 349, total_revenue:  14540.99, avg_transaction:   41.66 },
  { category: 'SUBSCRIPTIONS',  total_transactions: 327, total_revenue:   8464.81, avg_transaction:   25.89 },
];

const revenueByProvince = [
  { province: 'SK', total_transactions: 737, total_revenue: 360187.06, avg_transaction: 488.72, unique_customers: 43 },
  { province: 'ON', total_transactions: 663, total_revenue: 329424.24, avg_transaction: 496.87, unique_customers: 38 },
  { province: 'AB', total_transactions: 502, total_revenue: 253216.97, avg_transaction: 504.42, unique_customers: 29 },
  { province: 'MB', total_transactions: 535, total_revenue: 239089.38, avg_transaction: 446.90, unique_customers: 34 },
  { province: 'BC', total_transactions: 507, total_revenue: 221273.04, avg_transaction: 436.44, unique_customers: 31 },
  { province: 'QC', total_transactions: 429, total_revenue: 194709.18, avg_transaction: 453.87, unique_customers: 25 },
];

const statusData = [
  { status: 'COMPLETED', count: 4162, pct: 83.24, color: '#00ff9f' },
  { status: 'PENDING',   count: 625,  pct: 12.50, color: '#f0a832' },
  { status: 'FAILED',    count: 213,  pct: 4.26,  color: '#ff4466' },
];

const C = {
  bg: '#030508', panel: '#060b10', panel2: '#090f16',
  border: 'rgba(0,255,159,0.12)', border2: 'rgba(0,255,159,0.28)',
  green: '#00ff9f', blue: '#00cfff', amber: '#f0a832',
  red: '#ff4466', purple: '#bf7fff', text: '#c8dce8', dim: '#4a6070', dimmer: '#253040',
};

const PROV_COLORS = ['#00ff9f','#00cfff','#f0a832','#bf7fff','#ff9f44','#ff4466'];
const CAT_COLORS  = ['#00ff9f','#00cfff','#f0a832','#bf7fff','#ff9f44','#ff4466','#44ffdd','#ffdd44','#ff44aa','#aaffee'];
const SEG_COLORS  = { Champions: '#00ff9f', Loyal: '#00cfff', 'At-Risk': '#f0a832', Lost: '#ff4466' };

const fmt  = n => `$${Number(n).toLocaleString('en-CA', { maximumFractionDigits: 0 })}`;
const fmtK = n => `$${(n / 1000).toFixed(1)}k`;

function useCounter(target, duration = 1200) {
  const [val, setVal] = useState(0);
  useEffect(() => {
    let start = null;
    const step = (ts) => {
      if (!start) start = ts;
      const progress = Math.min((ts - start) / duration, 1);
      const ease = 1 - Math.pow(1 - progress, 3);
      setVal(Math.floor(ease * target));
      if (progress < 1) requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  }, [target, duration]);
  return val;
}

const Tip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null;
  return (
    <div style={{ background: C.panel2, border: `1px solid ${C.border2}`, padding: '10px 14px', fontSize: 11, fontFamily: 'IBM Plex Mono, monospace' }}>
      <div style={{ color: C.green, marginBottom: 6 }}>{label}</div>
      {payload.map((p, i) => (
        <div key={i} style={{ color: p.color || C.text, marginBottom: 2 }}>
          {p.name}: <b style={{ color: '#fff' }}>{typeof p.value === 'number' && p.value > 500 ? fmt(p.value) : p.value?.toLocaleString()}</b>
        </div>
      ))}
    </div>
  );
};

const StatRow = ({ label, value, bar, barColor, sub }) => (
  <div style={{ padding: '10px 0', borderBottom: `1px solid ${C.dimmer}`, display: 'flex', alignItems: 'center', gap: 12 }}>
    <div style={{ width: 120, color: C.dim, fontSize: 10, letterSpacing: '0.08em', flexShrink: 0 }}>{label}</div>
    <div style={{ flex: 1 }}>
      {bar !== undefined && (
        <div style={{ height: 3, background: C.dimmer, borderRadius: 2, marginBottom: 4, overflow: 'hidden' }}>
          <div style={{ height: '100%', width: `${bar}%`, background: barColor || C.green, borderRadius: 2 }} />
        </div>
      )}
      <div style={{ color: '#fff', fontSize: 12, fontWeight: 700 }}>{value}</div>
      {sub && <div style={{ color: C.dim, fontSize: 10, marginTop: 2 }}>{sub}</div>}
    </div>
  </div>
);

const KpiCard = ({ label, rawValue, prefix = '', suffix = '', color = C.green, sub }) => {
  const animated = useCounter(rawValue);
  return (
    <div style={{ background: C.panel, border: `1px solid ${C.border}`, borderTop: `2px solid ${color}`, padding: '16px 20px', flex: 1, minWidth: 140, position: 'relative', overflow: 'hidden' }}>
      <div style={{ position: 'absolute', top: 0, right: 0, width: 60, height: 60, background: `radial-gradient(circle at top right, ${color}18, transparent 70%)`, pointerEvents: 'none' }} />
      <div style={{ color: C.dim, fontSize: 9, letterSpacing: '0.18em', textTransform: 'uppercase', marginBottom: 10 }}>{label}</div>
      <div style={{ color, fontSize: 22, fontWeight: 800, fontFamily: 'IBM Plex Mono, monospace', letterSpacing: '-0.02em' }}>
        {prefix}{animated.toLocaleString('en-CA')}{suffix}
      </div>
      {sub && <div style={{ color: C.dim, fontSize: 10, marginTop: 6 }}>{sub}</div>}
    </div>
  );
};

const globalStyle = `
  @import url('https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@300;400;500;600;700&family=Syne:wght@700;800&display=swap');
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { background: #030508; }
  ::-webkit-scrollbar { width: 4px; }
  ::-webkit-scrollbar-track { background: #060b10; }
  ::-webkit-scrollbar-thumb { background: #00ff9f44; border-radius: 2px; }
  @keyframes blink { 50% { opacity: 0; } }
  @keyframes fadeUp { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
  @keyframes spin { to { transform: rotate(360deg); } }
  .panel { animation: fadeUp 0.4s ease both; }
  .spinner { animation: spin 1s linear infinite; display: inline-block; }
`;

export default function App() {
  const [tab, setTab] = useState('OVERVIEW');
  const [predictData, setPredictData] = useState(null);
  const [predictLoading, setPredictLoading] = useState(false);
  const [insightQuestion, setInsightQuestion] = useState('');
  const [insightAnswer, setInsightAnswer] = useState(null);
  const [insightLoading, setInsightLoading] = useState(false);
  const [insightReport, setInsightReport] = useState(null);
  const [reportLoading, setReportLoading] = useState(false);

  const totalRevenue = monthlyRevenue.reduce((s, r) => s + r.total_revenue, 0);
  const avgMonthly = totalRevenue / 12;
  const tabs = ['OVERVIEW', 'MONTHLY', 'CATEGORY', 'PROVINCE', 'PREDICTIONS', 'INSIGHTS', 'PIPELINE'];

  // Load predictions when tab opens
  useEffect(() => {
    if (tab === 'PREDICTIONS' && !predictData) {
      setPredictLoading(true);
      Promise.all([
        fetch(`${BASE_URL}/predict/forecast`).then(r => r.json()),
        fetch(`${BASE_URL}/predict/churn`).then(r => r.json()),
        fetch(`${BASE_URL}/predict/segments`).then(r => r.json()),
        fetch(`${BASE_URL}/predict/fraud`).then(r => r.json()),
      ]).then(([forecast, churn, segments, fraud]) => {
        setPredictData({ forecast: forecast.forecast, churn: churn.churn, segments: segments.segments, fraud: fraud.fraud });
        setPredictLoading(false);
      }).catch(() => setPredictLoading(false));
    }
  }, [tab, predictData]);

  const askInsight = async () => {
    if (!insightQuestion.trim()) return;
    setInsightLoading(true);
    setInsightAnswer(null);
    try {
      const res = await fetch(`${BASE_URL}/insights/ask?q=${encodeURIComponent(insightQuestion)}`);
      const data = await res.json();
      setInsightAnswer(data.result);
    } catch (e) {
      setInsightAnswer({ error: 'Failed to get insight' });
    }
    setInsightLoading(false);
  };

  const loadReport = async () => {
    setReportLoading(true);
    try {
      const res = await fetch(`${BASE_URL}/insights/report`);
      const data = await res.json();
      setInsightReport(data.result?.report);
    } catch (e) {}
    setReportLoading(false);
  };

  const QUICK_QUESTIONS = [
    'Why did revenue drop in September?',
    'Which province is performing best?',
    'What is our top revenue category?',
    'What should we focus on next quarter?',
    'Which customers are at risk of churning?',
  ];

  return (
    <>
      <style>{globalStyle}</style>
      <div style={{ background: C.bg, minHeight: '100vh', color: C.text, fontFamily: 'IBM Plex Mono, monospace', fontSize: 12 }}>
        <div style={{ borderBottom: `1px solid ${C.border}`, padding: '0 32px', display: 'flex', alignItems: 'center', height: 48, background: 'linear-gradient(180deg, #060d14 0%, transparent 100%)', position: 'sticky', top: 0, zIndex: 100, backdropFilter: 'blur(8px)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1 }}>
            <div style={{ width: 6, height: 6, borderRadius: '50%', background: C.green, boxShadow: `0 0 8px ${C.green}`, animation: 'blink 2s step-end infinite' }} />
            <span style={{ color: C.green, fontSize: 10, letterSpacing: '0.2em' }}>FINTECH-SAAS-PIPELINE</span>
            <span style={{ color: C.dimmer }}>|</span>
            <span style={{ color: C.dim, fontSize: 10 }}>DBT + POSTGRESQL + AI</span>
          </div>
          <div style={{ display: 'flex', gap: 24, fontSize: 10, color: C.dim }}>
            <span>5,000 TXN</span><span style={{ color: C.border2 }}>|</span>
            <span>6 PROVINCES</span><span style={{ color: C.border2 }}>|</span>
            <span>10 CATEGORIES</span><span style={{ color: C.border2 }}>|</span>
            <span style={{ color: C.green }}>16/16 TESTS ✓</span><span style={{ color: C.border2 }}>|</span>
            <span style={{ color: C.purple }}>AI INSIGHTS ✓</span>
          </div>
        </div>

        <div style={{ padding: '28px 32px', maxWidth: 1280, margin: '0 auto' }}>
          <div style={{ marginBottom: 28, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
            <div>
              <div style={{ fontSize: 9, letterSpacing: '0.25em', color: C.dim, marginBottom: 6 }}>// ANALYTICS DASHBOARD · 2025 · CANADA</div>
              <h1 style={{ fontFamily: 'Syne, sans-serif', fontSize: 36, fontWeight: 800, color: '#fff', letterSpacing: '-0.03em', lineHeight: 1 }}>
                FinTech <span style={{ color: C.green }}>Analytics</span>
              </h1>
            </div>
            <div style={{ textAlign: 'right', fontSize: 10, color: C.dim }}>
              <div style={{ color: C.green, fontSize: 18, fontWeight: 700, fontFamily: 'IBM Plex Mono' }}>{fmt(totalRevenue)}</div>
              <div style={{ letterSpacing: '0.1em' }}>TOTAL REVENUE · CAD 2025</div>
            </div>
          </div>

          <div style={{ display: 'flex', gap: 10, marginBottom: 24, flexWrap: 'wrap' }}>
            <KpiCard label="Total Revenue" rawValue={Math.round(totalRevenue)} prefix="$" color={C.green} sub="CAD · all provinces 2025" />
            <KpiCard label="Transactions" rawValue={5000} color={C.blue} sub="83.2% completion rate" />
            <KpiCard label="Avg Monthly" rawValue={Math.round(avgMonthly)} prefix="$" color={C.amber} sub="peak: MAR $157K" />
            <KpiCard label="Provinces" rawValue={6} color={C.purple} sub="SK · ON · AB · MB · BC · QC" />
            <KpiCard label="Categories" rawValue={10} color={C.red} sub="transfers dominates 55%" />
          </div>

          <div style={{ display: 'flex', gap: 0, marginBottom: 20, borderBottom: `1px solid ${C.dimmer}` }}>
            {tabs.map(t => (
              <button key={t} onClick={() => setTab(t)} style={{
                background: 'none', border: 'none', cursor: 'pointer', padding: '10px 18px',
                fontSize: 10, letterSpacing: '0.15em', fontFamily: 'IBM Plex Mono, monospace', fontWeight: 600,
                color: tab === t ? C.green : C.dim,
                borderBottom: tab === t ? `2px solid ${C.green}` : '2px solid transparent',
                marginBottom: -1, transition: 'color .15s',
                textShadow: tab === t ? `0 0 12px ${C.green}` : 'none',
              }}>{t}</button>
            ))}
          </div>

          {/* ── OVERVIEW ── */}
          {tab === 'OVERVIEW' && (
            <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 16 }}>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 20 }}>── MONTHLY REVENUE TREND · CAD 2025</div>
                <ResponsiveContainer width="100%" height={220}>
                  <AreaChart data={monthlyRevenue}>
                    <defs>
                      <linearGradient id="g1" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor={C.green} stopOpacity={0.25} />
                        <stop offset="95%" stopColor={C.green} stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} />
                    <XAxis dataKey="m" tick={{ fill: C.dim, fontSize: 9 }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fill: C.dim, fontSize: 9 }} tickFormatter={fmtK} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <ReferenceLine y={avgMonthly} stroke={C.amber} strokeDasharray="4 4" label={{ value: 'AVG', fill: C.amber, fontSize: 9 }} />
                    <Area type="monotone" dataKey="total_revenue" name="Revenue" stroke={C.green} fill="url(#g1)" strokeWidth={2} dot={{ r: 3, fill: C.green, strokeWidth: 0 }} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 20 }}>── TRANSACTION STATUS</div>
                <ResponsiveContainer width="100%" height={160}>
                  <PieChart>
                    <Pie data={statusData} dataKey="count" nameKey="status" cx="50%" cy="50%" innerRadius={45} outerRadius={70} paddingAngle={3}>
                      {statusData.map((s, i) => <Cell key={i} fill={s.color} />)}
                    </Pie>
                    <Tooltip formatter={(v, n) => [v.toLocaleString(), n]} contentStyle={{ background: C.panel2, border: `1px solid ${C.border2}`, fontSize: 11 }} />
                  </PieChart>
                </ResponsiveContainer>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6, marginTop: 8 }}>
                  {statusData.map((s, i) => (
                    <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <div style={{ width: 8, height: 8, borderRadius: 1, background: s.color }} />
                      <span style={{ color: C.dim, fontSize: 9, flex: 1 }}>{s.status}</span>
                      <span style={{ color: '#fff', fontSize: 10, fontWeight: 700 }}>{s.pct}%</span>
                    </div>
                  ))}
                </div>
              </div>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.blue, marginBottom: 20 }}>── REVENUE BY PROVINCE</div>
                <ResponsiveContainer width="100%" height={200}>
                  <BarChart data={revenueByProvince} barSize={28}>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} vertical={false} />
                    <XAxis dataKey="province" tick={{ fill: C.dim, fontSize: 10 }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fill: C.dim, fontSize: 9 }} tickFormatter={fmtK} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <Bar dataKey="total_revenue" name="Revenue" radius={[2, 2, 0, 0]}>
                      {revenueByProvince.map((_, i) => <Cell key={i} fill={PROV_COLORS[i]} />)}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.amber, marginBottom: 16 }}>── TOP CATEGORIES</div>
                {revenueByCategory.slice(0, 5).map((c, i) => (
                  <StatRow key={i} label={c.category} value={fmt(c.total_revenue)}
                    bar={(c.total_revenue / revenueByCategory[0].total_revenue) * 100}
                    barColor={CAT_COLORS[i]} sub={`${c.total_transactions} txn · avg ${fmt(c.avg_transaction)}`} />
                ))}
              </div>
            </div>
          )}

          {/* ── MONTHLY ── */}
          {tab === 'MONTHLY' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 20 }}>── MONTHLY TREND · fct_monthly_revenue</div>
                <ResponsiveContainer width="100%" height={280}>
                  <LineChart data={monthlyRevenue}>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} />
                    <XAxis dataKey="m" tick={{ fill: C.dim, fontSize: 9 }} axisLine={false} tickLine={false} />
                    <YAxis yAxisId="l" tick={{ fill: C.dim, fontSize: 9 }} tickFormatter={fmtK} axisLine={false} tickLine={false} />
                    <YAxis yAxisId="r" orientation="right" tick={{ fill: C.dim, fontSize: 9 }} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <Line yAxisId="l" type="monotone" dataKey="total_revenue" name="Revenue" stroke={C.green} strokeWidth={2} dot={{ r: 3, fill: C.green }} />
                    <Line yAxisId="r" type="monotone" dataKey="total_transactions" name="Transactions" stroke={C.amber} strokeWidth={1.5} strokeDasharray="6 3" dot={{ r: 2 }} />
                    <Line yAxisId="r" type="monotone" dataKey="unique_customers" name="Customers" stroke={C.purple} strokeWidth={1.5} strokeDasharray="2 4" dot={{ r: 2 }} />
                  </LineChart>
                </ResponsiveContainer>
              </div>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 16 }}>── MONTHLY DATA TABLE</div>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr>{['MONTH','REVENUE','TRANSACTIONS','AVG TXN','CUSTOMERS','Δ MOM'].map(h => (
                      <th key={h} style={{ textAlign: 'left', padding: '6px 12px', color: C.dim, fontSize: 9, letterSpacing: '0.12em', borderBottom: `1px solid ${C.dimmer}` }}>{h}</th>
                    ))}</tr>
                  </thead>
                  <tbody>
                    {monthlyRevenue.map((r, i) => {
                      const prev = monthlyRevenue[i - 1];
                      const mom = prev ? ((r.total_revenue - prev.total_revenue) / prev.total_revenue * 100) : null;
                      return (
                        <tr key={i} style={{ borderBottom: `1px solid ${C.dimmer}` }}
                          onMouseEnter={e => e.currentTarget.style.background = C.panel2}
                          onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
                          <td style={{ padding: '9px 12px', color: C.green, fontWeight: 700 }}>{r.m}</td>
                          <td style={{ padding: '9px 12px', color: '#fff', fontWeight: 600 }}>{fmt(r.total_revenue)}</td>
                          <td style={{ padding: '9px 12px', color: C.text }}>{r.total_transactions}</td>
                          <td style={{ padding: '9px 12px', color: C.text }}>{fmt(r.avg_transaction)}</td>
                          <td style={{ padding: '9px 12px', color: C.text }}>{r.unique_customers}</td>
                          <td style={{ padding: '9px 12px', color: mom === null ? C.dim : mom >= 0 ? C.green : C.red, fontWeight: 600 }}>
                            {mom === null ? '—' : `${mom >= 0 ? '+' : ''}${mom.toFixed(1)}%`}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* ── CATEGORY ── */}
          {tab === 'CATEGORY' && (
            <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: 16 }}>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.amber, marginBottom: 20 }}>── REVENUE BY CATEGORY · fct_revenue_by_category</div>
                <ResponsiveContainer width="100%" height={340}>
                  <BarChart data={revenueByCategory} layout="vertical" barSize={16}>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} horizontal={false} />
                    <XAxis type="number" tick={{ fill: C.dim, fontSize: 9 }} tickFormatter={fmtK} axisLine={false} tickLine={false} />
                    <YAxis dataKey="category" type="category" tick={{ fill: C.text, fontSize: 9 }} width={100} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <Bar dataKey="total_revenue" name="Revenue" radius={[0, 2, 2, 0]}>
                      {revenueByCategory.map((_, i) => <Cell key={i} fill={CAT_COLORS[i]} />)}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.amber, marginBottom: 16 }}>── CATEGORY BREAKDOWN</div>
                {revenueByCategory.map((c, i) => {
                  const total = revenueByCategory.reduce((s, r) => s + r.total_revenue, 0);
                  const pct = ((c.total_revenue / total) * 100).toFixed(1);
                  return (
                    <div key={i} style={{ padding: '8px 0', borderBottom: `1px solid ${C.dimmer}` }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          <div style={{ width: 6, height: 6, borderRadius: 1, background: CAT_COLORS[i] }} />
                          <span style={{ color: C.text, fontSize: 10 }}>{c.category}</span>
                        </div>
                        <span style={{ color: '#fff', fontSize: 10, fontWeight: 700 }}>{pct}%</span>
                      </div>
                      <div style={{ height: 3, background: C.dimmer, borderRadius: 2, overflow: 'hidden' }}>
                        <div style={{ height: '100%', width: `${pct}%`, background: CAT_COLORS[i], borderRadius: 2 }} />
                      </div>
                      <div style={{ color: C.dim, fontSize: 9, marginTop: 3 }}>{fmt(c.total_revenue)} · {c.total_transactions} txn</div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* ── PROVINCE ── */}
          {tab === 'PROVINCE' && (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.blue, marginBottom: 20 }}>── REVENUE BY PROVINCE</div>
                <ResponsiveContainer width="100%" height={260}>
                  <BarChart data={revenueByProvince} barSize={36}>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} vertical={false} />
                    <XAxis dataKey="province" tick={{ fill: C.dim, fontSize: 10 }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fill: C.dim, fontSize: 9 }} tickFormatter={fmtK} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <Bar dataKey="total_revenue" name="Revenue" radius={[3, 3, 0, 0]}>
                      {revenueByProvince.map((_, i) => <Cell key={i} fill={PROV_COLORS[i]} />)}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.blue, marginBottom: 16 }}>── PROVINCE DETAIL</div>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr>{['PROVINCE','REVENUE','TXN','CUSTOMERS'].map(h => (
                      <th key={h} style={{ textAlign: 'left', padding: '6px 10px', color: C.dim, fontSize: 9, borderBottom: `1px solid ${C.dimmer}` }}>{h}</th>
                    ))}</tr>
                  </thead>
                  <tbody>
                    {revenueByProvince.map((r, i) => (
                      <tr key={i} style={{ borderBottom: `1px solid ${C.dimmer}` }}
                        onMouseEnter={e => e.currentTarget.style.background = C.panel2}
                        onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
                        <td style={{ padding: '9px 10px' }}>
                          <span style={{ background: PROV_COLORS[i], color: '#000', fontWeight: 800, fontSize: 10, padding: '2px 6px', borderRadius: 2 }}>{r.province}</span>
                        </td>
                        <td style={{ padding: '9px 10px', color: '#fff', fontWeight: 600 }}>{fmt(r.total_revenue)}</td>
                        <td style={{ padding: '9px 10px', color: C.text }}>{r.total_transactions}</td>
                        <td style={{ padding: '9px 10px', color: C.text }}>{r.unique_customers}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* ── PREDICTIONS ── */}
          {tab === 'PREDICTIONS' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {predictLoading && (
                <div style={{ textAlign: 'center', padding: 60, color: C.green }}>
                  <div className="spinner" style={{ fontSize: 24, marginBottom: 12 }}>⟳</div>
                  <div style={{ fontSize: 11, letterSpacing: '0.15em' }}>LOADING AI PREDICTIONS...</div>
                </div>
              )}
              {predictData && (
                <>
                  {/* Revenue Forecast */}
                  <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                    <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 20 }}>── REVENUE FORECAST · NEXT 3 MONTHS · linear_trend_seasonal</div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: 12, marginBottom: 20 }}>
                      {predictData.forecast?.forecasts?.map((f, i) => (
                        <div key={i} style={{ background: C.panel2, border: `1px solid ${C.border}`, borderTop: `2px solid ${C.green}`, padding: 16 }}>
                          <div style={{ color: C.dim, fontSize: 9, letterSpacing: '0.15em', marginBottom: 8 }}>{f.month}</div>
                          <div style={{ color: C.green, fontSize: 20, fontWeight: 700 }}>{fmt(f.forecast)}</div>
                          <div style={{ color: C.dim, fontSize: 9, marginTop: 6 }}>
                            Range: {fmt(f.lower)} – {fmt(f.upper)}
                          </div>
                          <div style={{ color: C.dim, fontSize: 9 }}>Seasonal index: {f.seasonal_index}</div>
                        </div>
                      ))}
                    </div>
                    <div style={{ display: 'flex', gap: 24, fontSize: 10 }}>
                      <span style={{ color: C.dim }}>Trend: <span style={{ color: predictData.forecast?.trend_dir === 'up' ? C.green : C.red }}>{predictData.forecast?.trend_pct}% {predictData.forecast?.trend_dir}</span></span>
                      <span style={{ color: C.dim }}>Avg monthly: <span style={{ color: '#fff' }}>{fmt(predictData.forecast?.avg_monthly)}</span></span>
                      <span style={{ color: C.dim }}>Data points: <span style={{ color: '#fff' }}>{predictData.forecast?.data_points}</span></span>
                    </div>
                  </div>

                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                    {/* Churn */}
                    <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                      <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.red, marginBottom: 16 }}>── CHURN RISK · rule_based_churn_scorer</div>
                      <div style={{ display: 'flex', gap: 16, marginBottom: 16 }}>
                        <div style={{ background: C.panel2, border: `1px solid ${C.red}44`, borderRadius: 6, padding: '10px 16px', flex: 1, textAlign: 'center' }}>
                          <div style={{ color: C.red, fontSize: 20, fontWeight: 700 }}>{predictData.churn?.churn_rate}%</div>
                          <div style={{ color: C.dim, fontSize: 9 }}>CHURN RATE</div>
                        </div>
                        <div style={{ background: C.panel2, border: `1px solid ${C.red}44`, borderRadius: 6, padding: '10px 16px', flex: 1, textAlign: 'center' }}>
                          <div style={{ color: C.red, fontSize: 20, fontWeight: 700 }}>{predictData.churn?.high_risk}</div>
                          <div style={{ color: C.dim, fontSize: 9 }}>HIGH RISK</div>
                        </div>
                        <div style={{ background: C.panel2, border: `1px solid ${C.amber}44`, borderRadius: 6, padding: '10px 16px', flex: 1, textAlign: 'center' }}>
                          <div style={{ color: C.amber, fontSize: 20, fontWeight: 700 }}>{predictData.churn?.medium_risk}</div>
                          <div style={{ color: C.dim, fontSize: 9 }}>MEDIUM RISK</div>
                        </div>
                      </div>
                      {predictData.churn?.top_at_risk?.filter(c => c.churn_score > 0).map((c, i) => (
                        <div key={i} style={{ padding: '8px 0', borderBottom: `1px solid ${C.dimmer}` }}>
                          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                            <span style={{ color: C.text, fontSize: 11 }}>{c.customer_ref}</span>
                            <span style={{ color: c.risk_level === 'HIGH' ? C.red : C.amber, fontSize: 10, fontWeight: 700 }}>{c.risk_level} · {c.churn_score}</span>
                          </div>
                          <div style={{ color: C.dim, fontSize: 9 }}>{c.reasons.join(' · ')}</div>
                        </div>
                      ))}
                    </div>

                    {/* Segments */}
                    <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                      <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.purple, marginBottom: 16 }}>── CUSTOMER SEGMENTS · RFM SCORING</div>
                      {predictData.segments?.segments && Object.entries(predictData.segments.segments).map(([seg, data], i) => (
                        <div key={i} style={{ padding: '10px 0', borderBottom: `1px solid ${C.dimmer}` }}>
                          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              <div style={{ width: 8, height: 8, borderRadius: 1, background: SEG_COLORS[seg] || C.purple }} />
                              <span style={{ color: '#fff', fontWeight: 700, fontSize: 12 }}>{seg}</span>
                            </div>
                            <span style={{ color: SEG_COLORS[seg] || C.purple, fontSize: 11, fontWeight: 700 }}>{data.pct}%</span>
                          </div>
                          <div style={{ height: 4, background: C.dimmer, borderRadius: 2, overflow: 'hidden', marginBottom: 4 }}>
                            <div style={{ height: '100%', width: `${data.pct}%`, background: SEG_COLORS[seg] || C.purple, borderRadius: 2 }} />
                          </div>
                          <div style={{ color: C.dim, fontSize: 9 }}>{data.count} customers · {fmt(data.total_spend)} spend · avg RFM {data.avg_rfm}</div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Fraud */}
                  <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                    <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.amber, marginBottom: 16 }}>── FRAUD DETECTION · statistical_anomaly_detector</div>
                    <div style={{ display: 'flex', gap: 16, marginBottom: 16 }}>
                      <div style={{ background: C.panel2, border: `1px solid ${C.amber}44`, borderRadius: 6, padding: '10px 16px' }}>
                        <div style={{ color: C.amber, fontSize: 18, fontWeight: 700 }}>{predictData.fraud?.flagged}</div>
                        <div style={{ color: C.dim, fontSize: 9 }}>FLAGGED</div>
                      </div>
                      <div style={{ background: C.panel2, border: `1px solid ${C.border}`, borderRadius: 6, padding: '10px 16px' }}>
                        <div style={{ color: '#fff', fontSize: 18, fontWeight: 700 }}>{predictData.fraud?.total_analyzed}</div>
                        <div style={{ color: C.dim, fontSize: 9 }}>ANALYZED</div>
                      </div>
                      <div style={{ background: C.panel2, border: `1px solid ${C.border}`, borderRadius: 6, padding: '10px 16px' }}>
                        <div style={{ color: C.amber, fontSize: 18, fontWeight: 700 }}>{predictData.fraud?.flag_rate}%</div>
                        <div style={{ color: C.dim, fontSize: 9 }}>FLAG RATE</div>
                      </div>
                      <div style={{ background: C.panel2, border: `1px solid ${C.border}`, borderRadius: 6, padding: '10px 16px', flex: 1 }}>
                        <div style={{ color: C.dim, fontSize: 9, marginBottom: 4 }}>STATS · mean {fmt(predictData.fraud?.stats?.mean)} · stdev {fmt(predictData.fraud?.stats?.stdev)}</div>
                        <div style={{ color: C.dim, fontSize: 9 }}>Q1 {fmt(predictData.fraud?.stats?.q1)} · Q3 {fmt(predictData.fraud?.stats?.q3)} · IQR {fmt(predictData.fraud?.stats?.iqr)}</div>
                      </div>
                    </div>
                    {predictData.fraud?.top_flagged?.map((f, i) => (
                      <div key={i} style={{ padding: '8px 14px', marginBottom: 6, background: C.panel2, border: `1px solid ${f.risk_level === 'HIGH' ? C.red : C.amber}44`, borderLeft: `2px solid ${f.risk_level === 'HIGH' ? C.red : C.amber}`, borderRadius: 4, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span style={{ color: C.text, fontSize: 11 }}>{f.transaction_id}</span>
                        <span style={{ color: '#fff', fontWeight: 700 }}>{fmt(f.amount)}</span>
                        <span style={{ color: C.dim, fontSize: 10 }}>Z-score {f.z_score}σ</span>
                        <span style={{ color: f.risk_level === 'HIGH' ? C.red : C.amber, fontSize: 10, fontWeight: 700 }}>{f.risk_level}</span>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </div>
          )}

          {/* ── INSIGHTS ── */}
          {tab === 'INSIGHTS' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {/* Ask anything */}
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.purple, marginBottom: 20 }}>── ASK ANYTHING · RAG + CLAUDE API · grounded in your data</div>
                <div style={{ display: 'flex', gap: 10, marginBottom: 16 }}>
                  <input
                    value={insightQuestion}
                    onChange={e => setInsightQuestion(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && askInsight()}
                    placeholder="Ask a question about your business data..."
                    style={{
                      flex: 1, background: C.panel2, border: `1px solid ${C.border2}`,
                      borderRadius: 4, padding: '10px 14px', color: C.text,
                      fontFamily: 'IBM Plex Mono, monospace', fontSize: 12, outline: 'none',
                    }}
                  />
                  <button onClick={askInsight} disabled={insightLoading} style={{
                    background: C.green, color: '#000', border: 'none', borderRadius: 4,
                    padding: '10px 20px', fontFamily: 'IBM Plex Mono, monospace',
                    fontSize: 11, fontWeight: 700, cursor: 'pointer', letterSpacing: '0.1em',
                    opacity: insightLoading ? 0.6 : 1,
                  }}>
                    {insightLoading ? '...' : 'ASK →'}
                  </button>
                </div>
                {/* Quick questions */}
                <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 20 }}>
                  {QUICK_QUESTIONS.map((q, i) => (
                    <button key={i} onClick={() => { setInsightQuestion(q); }}
                      style={{
                        background: 'transparent', border: `1px solid ${C.border2}`,
                        borderRadius: 20, padding: '4px 12px', color: C.dim,
                        fontFamily: 'IBM Plex Mono, monospace', fontSize: 10, cursor: 'pointer',
                        transition: 'all .15s',
                      }}
                      onMouseEnter={e => { e.target.style.color = C.green; e.target.style.borderColor = C.green; }}
                      onMouseLeave={e => { e.target.style.color = C.dim; e.target.style.borderColor = C.border2; }}>
                      {q}
                    </button>
                  ))}
                </div>
                {/* Answer */}
                {insightAnswer && (
                  <div style={{ background: C.panel2, border: `1px solid ${C.border2}`, borderRadius: 4, padding: 20 }}>
                    <div style={{ fontSize: 9, color: C.purple, letterSpacing: '0.15em', marginBottom: 12 }}>── AI ANSWER · grounded in dbt data</div>
                    <div style={{ color: C.text, fontSize: 12, lineHeight: 1.8, whiteSpace: 'pre-wrap' }}>
                      {insightAnswer.answer || insightAnswer.error}
                    </div>
                    {insightAnswer.ai_available === false && (
                      <div style={{ marginTop: 12, padding: 12, background: C.panel, borderRadius: 4, color: C.dim, fontSize: 10 }}>
                        <div style={{ color: C.amber, marginBottom: 6 }}>RAW DATA CONTEXT:</div>
                        <pre style={{ whiteSpace: 'pre-wrap', fontSize: 10, color: C.dim }}>{insightAnswer.context}</pre>
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* Executive Report */}
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
                  <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.blue }}>── EXECUTIVE REPORT · AI-GENERATED SUMMARY</div>
                  <button onClick={loadReport} disabled={reportLoading} style={{
                    background: 'transparent', border: `1px solid ${C.blue}`, borderRadius: 4,
                    padding: '6px 14px', color: C.blue, fontFamily: 'IBM Plex Mono, monospace',
                    fontSize: 10, cursor: 'pointer', letterSpacing: '0.1em',
                  }}>
                    {reportLoading ? 'GENERATING...' : 'GENERATE REPORT →'}
                  </button>
                </div>
                {insightReport && (
                  <div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,1fr)', gap: 10, marginBottom: 20 }}>
                      {[
                        { label: 'TOTAL REVENUE', value: fmt(insightReport.total_revenue), color: C.green },
                        { label: 'TRANSACTIONS', value: insightReport.total_transactions?.toLocaleString(), color: C.blue },
                        { label: 'COMPLETION', value: `${insightReport.completion_rate}%`, color: C.amber },
                        { label: 'TOP PROVINCE', value: insightReport.top_province, color: C.purple },
                      ].map((k, i) => (
                        <div key={i} style={{ background: C.panel2, border: `1px solid ${C.border}`, borderTop: `2px solid ${k.color}`, padding: '12px 14px' }}>
                          <div style={{ color: C.dim, fontSize: 9, marginBottom: 6 }}>{k.label}</div>
                          <div style={{ color: k.color, fontSize: 16, fontWeight: 700 }}>{k.value}</div>
                        </div>
                      ))}
                    </div>
                    <div style={{ background: C.panel2, border: `1px solid ${C.border2}`, borderRadius: 4, padding: 20 }}>
                      <div style={{ fontSize: 9, color: C.blue, letterSpacing: '0.15em', marginBottom: 12 }}>── AI EXECUTIVE SUMMARY · {insightReport.period}</div>
                      <div style={{ color: C.text, fontSize: 12, lineHeight: 1.8, whiteSpace: 'pre-wrap' }}>{insightReport.ai_summary}</div>
                      <div style={{ color: C.dim, fontSize: 9, marginTop: 12 }}>Generated: {new Date(insightReport.generated_at).toLocaleString()}</div>
                    </div>
                  </div>
                )}
                {!insightReport && !reportLoading && (
                  <div style={{ textAlign: 'center', padding: 40, color: C.dim, fontSize: 11 }}>
                    Click "GENERATE REPORT" to create an AI executive summary of your 2025 performance
                  </div>
                )}
              </div>
            </div>
          )}

          {/* ── PIPELINE ── */}
          {tab === 'PIPELINE' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 28 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 24 }}>── DATA PIPELINE ARCHITECTURE · END TO END</div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 4, alignItems: 'center', marginBottom: 32 }}>
                  {[
                    { l: 'CSV', s: 'upload', c: C.amber, icon: '📄' },
                    { l: '▶', s: '', c: C.dimmer, icon: '' },
                    { l: 'PIPELINE', s: 'Go service', c: C.green, icon: '⚙' },
                    { l: '▶', s: '', c: C.dimmer, icon: '' },
                    { l: 'POSTGRESQL', s: '5,000 rows', c: C.blue, icon: '🐘' },
                    { l: '▶', s: '', c: C.dimmer, icon: '' },
                    { l: 'DBT', s: '5 models · 16 tests', c: C.amber, icon: '⬟' },
                  ].map((item, i) => (
                    <div key={i} style={{ textAlign: 'center' }}>
                      {item.icon && <div style={{ fontSize: 22, marginBottom: 6 }}>{item.icon}</div>}
                      <div style={{ color: item.c, fontSize: 10, fontWeight: 700 }}>{item.l}</div>
                      {item.s && <div style={{ color: C.dim, fontSize: 9, marginTop: 3 }}>{item.s}</div>}
                    </div>
                  ))}
                </div>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.amber, marginBottom: 16 }}>── DBT MODELS</div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10, marginBottom: 24 }}>
                  {[
                    { name: 'stg_transactions', type: 'VIEW', desc: 'Clean + validate raw transactions', tests: 4 },
                    { name: 'fct_monthly_revenue', type: 'TABLE', desc: '12 months · revenue + customers', tests: 4 },
                    { name: 'fct_revenue_by_category', type: 'TABLE', desc: '10 categories · avg transaction', tests: 3 },
                    { name: 'fct_revenue_by_province', type: 'TABLE', desc: '6 provinces · unique customers', tests: 3 },
                    { name: 'fct_status_summary', type: 'TABLE', desc: 'completed · pending · failed', tests: 2 },
                    { name: 'TEST SUITE', type: 'TESTS', desc: 'not_null · unique · accepted_values', tests: 16 },
                  ].map((m, i) => (
                    <div key={i} style={{ background: C.panel2, border: `1px solid ${C.dimmer}`, borderLeft: `2px solid ${C.green}`, padding: '12px 16px' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                        <span style={{ color: C.green, fontSize: 10, fontWeight: 700 }}>{m.name}</span>
                        <span style={{ color: C.dim, fontSize: 8, background: C.dimmer, padding: '1px 6px', borderRadius: 2 }}>{m.type}</span>
                      </div>
                      <div style={{ color: C.text, fontSize: 10 }}>{m.desc}</div>
                      <div style={{ color: C.green, fontSize: 9, marginTop: 6 }}>✓ {m.tests} tests passing</div>
                    </div>
                  ))}
                </div>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 16 }}>── LIVE AZURE CONTAINER APPS · CANADACENTRAL</div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {[
                    { name: 'auth-service',     port: 8081, status: 'OK',       color: C.green },
                    { name: 'ai-service',        port: 8085, status: 'OK',       color: C.green },
                    { name: 'pipeline-service',  port: 8086, status: 'OK',       color: C.green },
                    { name: 'features-service',  port: 8082, status: 'OK',       color: C.green },
                    { name: 'monitor-service',   port: 8083, status: 'DEGRADED', color: C.amber },
                  ].map((s, i) => (
                    <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 14, background: C.panel2, border: `1px solid ${C.dimmer}`, padding: '10px 16px', borderLeft: `2px solid ${s.color}` }}>
                      <div style={{ width: 7, height: 7, borderRadius: '50%', background: s.color, boxShadow: `0 0 8px ${s.color}` }} />
                      <span style={{ color: '#fff', fontWeight: 700, fontSize: 11, minWidth: 160 }}>{s.name}</span>
                      <span style={{ color: C.dim, fontSize: 9, minWidth: 60 }}>:{s.port}</span>
                      <a href={`https://${s.name}.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health`}
                        target="_blank" rel="noreferrer"
                        style={{ color: s.color, fontSize: 9, textDecoration: 'none', flex: 1 }}>
                        {s.name}.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io ↗
                      </a>
                      <span style={{ color: s.color, fontSize: 9, fontWeight: 700 }}>{s.status}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          <div style={{ marginTop: 40, paddingTop: 16, borderTop: `1px solid ${C.dimmer}`, display: 'flex', justifyContent: 'space-between', fontSize: 9, color: C.dimmer }}>
            <span>FINTECH-SAAS-PIPELINE · charlesnet76 · Victoria BC Canada</span>
            <span>Phase 0-4 complete · dbt + PostgreSQL + Redis + Sentry + Claude AI</span>
          </div>
        </div>
      </div>
    </>
  );
}
