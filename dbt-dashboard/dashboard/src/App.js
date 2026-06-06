import { useState, useEffect, useRef } from 'react';
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  AreaChart, Area, ReferenceLine
} from 'recharts';

// ── Real data from PostgreSQL via dbt ────────────────────────────────────────
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

// ── Animated counter hook ─────────────────────────────────────────────────────
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

// ── Helpers ───────────────────────────────────────────────────────────────────
const C = {
  bg:      '#030508',
  panel:   '#060b10',
  panel2:  '#090f16',
  border:  'rgba(0,255,159,0.12)',
  border2: 'rgba(0,255,159,0.28)',
  green:   '#00ff9f',
  blue:    '#00cfff',
  amber:   '#f0a832',
  red:     '#ff4466',
  purple:  '#bf7fff',
  text:    '#c8dce8',
  dim:     '#4a6070',
  dimmer:  '#253040',
};

const PROV_COLORS = ['#00ff9f','#00cfff','#f0a832','#bf7fff','#ff9f44','#ff4466'];
const CAT_COLORS  = ['#00ff9f','#00cfff','#f0a832','#bf7fff','#ff9f44','#ff4466','#44ffdd','#ffdd44','#ff44aa','#aaffee'];

const fmt  = n => `$${Number(n).toLocaleString('en-CA', { maximumFractionDigits: 0 })}`;
const fmtK = n => `$${(n / 1000).toFixed(1)}k`;

// ── Custom tooltip ────────────────────────────────────────────────────────────
const Tip = ({ active, payload, label }) => {
  if (!active || !payload?.length) return null;
  return (
    <div style={{ background: C.panel2, border: `1px solid ${C.border2}`, padding: '10px 14px', fontSize: 11, fontFamily: 'IBM Plex Mono, monospace' }}>
      <div style={{ color: C.green, marginBottom: 6, letterSpacing: '0.1em' }}>{label}</div>
      {payload.map((p, i) => (
        <div key={i} style={{ color: p.color || C.text, marginBottom: 2 }}>
          {p.name}: <b style={{ color: '#fff' }}>{typeof p.value === 'number' && p.value > 500 ? fmt(p.value) : p.value?.toLocaleString()}</b>
        </div>
      ))}
    </div>
  );
};

// ── Stat row component ────────────────────────────────────────────────────────
const StatRow = ({ label, value, bar, barColor, sub }) => (
  <div style={{ padding: '10px 0', borderBottom: `1px solid ${C.dimmer}`, display: 'flex', alignItems: 'center', gap: 12 }}>
    <div style={{ width: 120, color: C.dim, fontSize: 10, letterSpacing: '0.08em', flexShrink: 0 }}>{label}</div>
    <div style={{ flex: 1 }}>
      {bar !== undefined && (
        <div style={{ height: 3, background: C.dimmer, borderRadius: 2, marginBottom: 4, overflow: 'hidden' }}>
          <div style={{ height: '100%', width: `${bar}%`, background: barColor || C.green, borderRadius: 2, boxShadow: `0 0 6px ${barColor || C.green}` }} />
        </div>
      )}
      <div style={{ color: '#fff', fontSize: 12, fontWeight: 700 }}>{value}</div>
      {sub && <div style={{ color: C.dim, fontSize: 10, marginTop: 2 }}>{sub}</div>}
    </div>
  </div>
);

// ── KPI card ──────────────────────────────────────────────────────────────────
const KpiCard = ({ label, rawValue, prefix = '', suffix = '', color = C.green, sub }) => {
  const animated = useCounter(rawValue);
  return (
    <div style={{
      background: C.panel, border: `1px solid ${C.border}`,
      borderTop: `2px solid ${color}`, padding: '16px 20px', flex: 1, minWidth: 140,
      position: 'relative', overflow: 'hidden',
    }}>
      <div style={{
        position: 'absolute', top: 0, right: 0, width: 60, height: 60,
        background: `radial-gradient(circle at top right, ${color}18, transparent 70%)`,
        pointerEvents: 'none',
      }} />
      <div style={{ color: C.dim, fontSize: 9, letterSpacing: '0.18em', textTransform: 'uppercase', marginBottom: 10 }}>{label}</div>
      <div style={{ color, fontSize: 22, fontWeight: 800, fontFamily: 'IBM Plex Mono, monospace', letterSpacing: '-0.02em' }}>
        {prefix}{animated.toLocaleString('en-CA')}{suffix}
      </div>
      {sub && <div style={{ color: C.dim, fontSize: 10, marginTop: 6 }}>{sub}</div>}
    </div>
  );
};

// ── Scanline overlay style ────────────────────────────────────────────────────
const globalStyle = `
  @import url('https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@300;400;500;600;700&family=Syne:wght@700;800&display=swap');
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { background: #030508; }
  ::-webkit-scrollbar { width: 4px; }
  ::-webkit-scrollbar-track { background: #060b10; }
  ::-webkit-scrollbar-thumb { background: #00ff9f44; border-radius: 2px; }
  @keyframes scanline {
    0% { transform: translateY(-100%); }
    100% { transform: translateY(100vh); }
  }
  @keyframes blink { 50% { opacity: 0; } }
  @keyframes fadeUp {
    from { opacity: 0; transform: translateY(12px); }
    to   { opacity: 1; transform: translateY(0); }
  }
  .panel { animation: fadeUp 0.4s ease both; }
  .panel:nth-child(2) { animation-delay: 0.05s; }
  .panel:nth-child(3) { animation-delay: 0.1s; }
  .panel:nth-child(4) { animation-delay: 0.15s; }
`;

// ── Main ──────────────────────────────────────────────────────────────────────
export default function App() {
  const [tab, setTab] = useState('OVERVIEW');
  const totalRevenue = monthlyRevenue.reduce((s, r) => s + r.total_revenue, 0);
  const avgMonthly = totalRevenue / 12;
  const tabs = ['OVERVIEW', 'MONTHLY', 'CATEGORY', 'PROVINCE', 'PIPELINE'];

  return (
    <>
      <style>{globalStyle}</style>
      <div style={{ background: C.bg, minHeight: '100vh', color: C.text, fontFamily: 'IBM Plex Mono, monospace', fontSize: 12 }}>

        {/* Scanline effect */}
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, pointerEvents: 'none', zIndex: 9999,
          background: 'repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.03) 2px, rgba(0,0,0,0.03) 4px)',
        }} />

        {/* Top bar */}
        <div style={{
          borderBottom: `1px solid ${C.border}`, padding: '0 32px',
          display: 'flex', alignItems: 'center', height: 48,
          background: 'linear-gradient(180deg, #060d14 0%, transparent 100%)',
          position: 'sticky', top: 0, zIndex: 100,
          backdropFilter: 'blur(8px)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1 }}>
            <div style={{ width: 6, height: 6, borderRadius: '50%', background: C.green, boxShadow: `0 0 8px ${C.green}`, animation: 'blink 2s step-end infinite' }} />
            <span style={{ color: C.green, fontSize: 10, letterSpacing: '0.2em' }}>FINTECH-SAAS-PIPELINE</span>
            <span style={{ color: C.dimmer }}>|</span>
            <span style={{ color: C.dim, fontSize: 10, letterSpacing: '0.12em' }}>DBT + POSTGRESQL + REACT</span>
          </div>
          <div style={{ display: 'flex', gap: 24, fontSize: 10, color: C.dim, letterSpacing: '0.1em' }}>
            <span>5,000 TXN</span>
            <span style={{ color: C.border2 }}>|</span>
            <span>6 PROVINCES</span>
            <span style={{ color: C.border2 }}>|</span>
            <span>10 CATEGORIES</span>
            <span style={{ color: C.border2 }}>|</span>
            <span style={{ color: C.green }}>16/16 TESTS ✓</span>
          </div>
        </div>

        <div style={{ padding: '28px 32px', maxWidth: 1280, margin: '0 auto' }}>

          {/* Title */}
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

          {/* KPI row */}
          <div style={{ display: 'flex', gap: 10, marginBottom: 24, flexWrap: 'wrap' }}>
            <KpiCard label="Total Revenue" rawValue={Math.round(totalRevenue)} prefix="$" color={C.green} sub="CAD · all provinces 2025" />
            <KpiCard label="Transactions" rawValue={5000} color={C.blue} sub="83.2% completion rate" />
            <KpiCard label="Avg Monthly" rawValue={Math.round(avgMonthly)} prefix="$" color={C.amber} sub="peak: MAR $157K" />
            <KpiCard label="Provinces" rawValue={6} color={C.purple} sub="SK · ON · AB · MB · BC · QC" />
            <KpiCard label="Categories" rawValue={10} color={C.red} sub="transfers dominates 55%" />
          </div>

          {/* Nav tabs */}
          <div style={{ display: 'flex', gap: 0, marginBottom: 20, borderBottom: `1px solid ${C.dimmer}` }}>
            {tabs.map(t => (
              <button key={t} onClick={() => setTab(t)} style={{
                background: 'none', border: 'none', cursor: 'pointer',
                padding: '10px 24px', fontSize: 10, letterSpacing: '0.15em',
                fontFamily: 'IBM Plex Mono, monospace', fontWeight: 600,
                color: tab === t ? C.green : C.dim,
                borderBottom: tab === t ? `2px solid ${C.green}` : '2px solid transparent',
                marginBottom: -1,
                transition: 'color .15s',
                textShadow: tab === t ? `0 0 12px ${C.green}` : 'none',
              }}>{t}</button>
            ))}
          </div>

          {/* ── OVERVIEW ── */}
          {tab === 'OVERVIEW' && (
            <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 16 }}>
              {/* Revenue area chart */}
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 20 }}>── MONTHLY REVENUE TREND · CAD 2025</div>
                <ResponsiveContainer width="100%" height={220}>
                  <AreaChart data={monthlyRevenue} margin={{ top: 5, right: 5, bottom: 0, left: 0 }}>
                    <defs>
                      <linearGradient id="g1" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%"  stopColor={C.green} stopOpacity={0.25} />
                        <stop offset="95%" stopColor={C.green} stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} />
                    <XAxis dataKey="m" tick={{ fill: C.dim, fontSize: 9 }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fill: C.dim, fontSize: 9 }} tickFormatter={fmtK} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <ReferenceLine y={avgMonthly} stroke={C.amber} strokeDasharray="4 4" label={{ value: 'AVG', fill: C.amber, fontSize: 9 }} />
                    <Area type="monotone" dataKey="total_revenue" name="Revenue" stroke={C.green} fill="url(#g1)" strokeWidth={2} dot={{ r: 3, fill: C.green, strokeWidth: 0 }} activeDot={{ r: 5, fill: C.green, boxShadow: `0 0 10px ${C.green}` }} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>

              {/* Status donut */}
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
                      <div style={{ width: 8, height: 8, borderRadius: 1, background: s.color, boxShadow: `0 0 6px ${s.color}` }} />
                      <span style={{ color: C.dim, fontSize: 9, flex: 1, letterSpacing: '0.08em' }}>{s.status}</span>
                      <span style={{ color: '#fff', fontSize: 10, fontWeight: 700 }}>{s.pct}%</span>
                      <span style={{ color: C.dim, fontSize: 9 }}>{s.count.toLocaleString()}</span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Province bars */}
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.blue, marginBottom: 20 }}>── REVENUE BY PROVINCE · fct_revenue_by_province</div>
                <ResponsiveContainer width="100%" height={200}>
                  <BarChart data={revenueByProvince} barSize={28}>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} vertical={false} />
                    <XAxis dataKey="province" tick={{ fill: C.dim, fontSize: 10 }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fill: C.dim, fontSize: 9 }} tickFormatter={fmtK} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <Bar dataKey="total_revenue" name="Revenue" radius={[2, 2, 0, 0]}>
                      {revenueByProvince.map((_, i) => (
                        <Cell key={i} fill={PROV_COLORS[i]} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>

              {/* Top categories */}
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.amber, marginBottom: 16 }}>── TOP CATEGORIES</div>
                {revenueByCategory.slice(0, 5).map((c, i) => (
                  <StatRow key={i}
                    label={c.category}
                    value={fmt(c.total_revenue)}
                    bar={(c.total_revenue / revenueByCategory[0].total_revenue) * 100}
                    barColor={CAT_COLORS[i]}
                    sub={`${c.total_transactions} txn · avg ${fmt(c.avg_transaction)}`}
                  />
                ))}
              </div>
            </div>
          )}

          {/* ── MONTHLY ── */}
          {tab === 'MONTHLY' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 20 }}>── MONTHLY TREND · fct_monthly_revenue · REVENUE + TRANSACTIONS + CUSTOMERS</div>
                <ResponsiveContainer width="100%" height={280}>
                  <LineChart data={monthlyRevenue}>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} />
                    <XAxis dataKey="m" tick={{ fill: C.dim, fontSize: 9 }} axisLine={false} tickLine={false} />
                    <YAxis yAxisId="l" tick={{ fill: C.dim, fontSize: 9 }} tickFormatter={fmtK} axisLine={false} tickLine={false} />
                    <YAxis yAxisId="r" orientation="right" tick={{ fill: C.dim, fontSize: 9 }} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <Line yAxisId="l" type="monotone" dataKey="total_revenue" name="Revenue" stroke={C.green} strokeWidth={2} dot={{ r: 3, fill: C.green }} />
                    <Line yAxisId="r" type="monotone" dataKey="total_transactions" name="Transactions" stroke={C.amber} strokeWidth={1.5} strokeDasharray="6 3" dot={{ r: 2, fill: C.amber }} />
                    <Line yAxisId="r" type="monotone" dataKey="unique_customers" name="Customers" stroke={C.purple} strokeWidth={1.5} strokeDasharray="2 4" dot={{ r: 2, fill: C.purple }} />
                  </LineChart>
                </ResponsiveContainer>
              </div>

              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 16 }}>── MONTHLY DATA TABLE</div>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr>
                      {['MONTH','REVENUE','TRANSACTIONS','AVG TXN','CUSTOMERS','Δ MOM'].map(h => (
                        <th key={h} style={{ textAlign: 'left', padding: '6px 12px', color: C.dim, fontSize: 9, letterSpacing: '0.12em', borderBottom: `1px solid ${C.dimmer}` }}>{h}</th>
                      ))}
                    </tr>
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
                  const totalCatRevenue = revenueByCategory.reduce((s, r) => s + r.total_revenue, 0);
                  const pct = ((c.total_revenue / totalCatRevenue) * 100).toFixed(1);
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
                        <div style={{ height: '100%', width: `${pct}%`, background: CAT_COLORS[i], borderRadius: 2, boxShadow: `0 0 6px ${CAT_COLORS[i]}66` }} />
                      </div>
                      <div style={{ color: C.dim, fontSize: 9, marginTop: 3 }}>{fmt(c.total_revenue)} · {c.total_transactions} txn · avg {fmt(c.avg_transaction)}</div>
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
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.blue, marginBottom: 20 }}>── TRANSACTIONS + CUSTOMERS</div>
                <ResponsiveContainer width="100%" height={260}>
                  <BarChart data={revenueByProvince} barSize={18}>
                    <CartesianGrid strokeDasharray="4 4" stroke={C.dimmer} vertical={false} />
                    <XAxis dataKey="province" tick={{ fill: C.dim, fontSize: 10 }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fill: C.dim, fontSize: 9 }} axisLine={false} tickLine={false} />
                    <Tooltip content={<Tip />} />
                    <Bar dataKey="total_transactions" name="Transactions" fill={C.blue} radius={[2, 2, 0, 0]} />
                    <Bar dataKey="unique_customers" name="Customers" fill={C.purple} radius={[2, 2, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>

              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24, gridColumn: 'span 2' }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.blue, marginBottom: 16 }}>── PROVINCE DETAIL · fct_revenue_by_province</div>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr>{['PROVINCE','REVENUE','TRANSACTIONS','AVG TXN','CUSTOMERS','REVENUE SHARE'].map(h => (
                      <th key={h} style={{ textAlign: 'left', padding: '6px 14px', color: C.dim, fontSize: 9, letterSpacing: '0.12em', borderBottom: `1px solid ${C.dimmer}` }}>{h}</th>
                    ))}</tr>
                  </thead>
                  <tbody>
                    {revenueByProvince.map((r, i) => {
                      const total = revenueByProvince.reduce((s, p) => s + p.total_revenue, 0);
                      const pct = ((r.total_revenue / total) * 100).toFixed(1);
                      return (
                        <tr key={i} style={{ borderBottom: `1px solid ${C.dimmer}` }}
                          onMouseEnter={e => e.currentTarget.style.background = C.panel2}
                          onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
                          <td style={{ padding: '10px 14px' }}>
                            <span style={{ background: PROV_COLORS[i], color: '#000', fontWeight: 800, fontSize: 10, padding: '2px 8px', borderRadius: 2 }}>{r.province}</span>
                          </td>
                          <td style={{ padding: '10px 14px', color: '#fff', fontWeight: 600 }}>{fmt(r.total_revenue)}</td>
                          <td style={{ padding: '10px 14px', color: C.text }}>{r.total_transactions}</td>
                          <td style={{ padding: '10px 14px', color: C.text }}>{fmt(r.avg_transaction)}</td>
                          <td style={{ padding: '10px 14px', color: C.text }}>{r.unique_customers}</td>
                          <td style={{ padding: '10px 14px' }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              <div style={{ height: 4, width: 80, background: C.dimmer, borderRadius: 2, overflow: 'hidden' }}>
                                <div style={{ height: '100%', width: `${pct}%`, background: PROV_COLORS[i], borderRadius: 2 }} />
                              </div>
                              <span style={{ color: PROV_COLORS[i], fontSize: 10, fontWeight: 700 }}>{pct}%</span>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* ── PIPELINE ── */}
          {tab === 'PIPELINE' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {/* Architecture flow */}
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
                      <div style={{ color: item.c, fontSize: 10, fontWeight: 700, letterSpacing: '0.08em' }}>{item.l}</div>
                      {item.s && <div style={{ color: C.dim, fontSize: 9, marginTop: 3 }}>{item.s}</div>}
                    </div>
                  ))}
                </div>

                {/* dbt models */}
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.amber, marginBottom: 16 }}>── DBT MODELS</div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
                  {[
                    { name: 'stg_transactions', type: 'VIEW', schema: 'analytics_staging', desc: 'Clean + validate raw transactions', tests: 4 },
                    { name: 'fct_monthly_revenue', type: 'TABLE', schema: 'analytics_marts', desc: '12 months · revenue + customers', tests: 4 },
                    { name: 'fct_revenue_by_category', type: 'TABLE', schema: 'analytics_marts', desc: '10 categories · avg transaction', tests: 3 },
                    { name: 'fct_revenue_by_province', type: 'TABLE', schema: 'analytics_marts', desc: '6 provinces · unique customers', tests: 3 },
                    { name: 'fct_status_summary', type: 'TABLE', schema: 'analytics_marts', desc: 'completed · pending · failed', tests: 2 },
                    { name: 'TEST SUITE', type: 'TESTS', schema: '', desc: 'not_null · unique · accepted_values', tests: 16 },
                  ].map((m, i) => (
                    <div key={i} style={{ background: C.panel2, border: `1px solid ${C.dimmer}`, borderLeft: `2px solid ${C.green}`, padding: '12px 16px' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                        <span style={{ color: C.green, fontSize: 10, fontWeight: 700 }}>{m.name}</span>
                        <span style={{ color: C.dim, fontSize: 8, background: C.dimmer, padding: '1px 6px', borderRadius: 2 }}>{m.type}</span>
                      </div>
                      {m.schema && <div style={{ color: C.dim, fontSize: 9, marginBottom: 4 }}>{m.schema}</div>}
                      <div style={{ color: C.text, fontSize: 10 }}>{m.desc}</div>
                      <div style={{ color: C.green, fontSize: 9, marginTop: 6 }}>✓ {m.tests} tests passing</div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Live services */}
              <div className="panel" style={{ background: C.panel, border: `1px solid ${C.border}`, padding: 24 }}>
                <div style={{ fontSize: 9, letterSpacing: '0.18em', color: C.green, marginBottom: 16 }}>── LIVE AZURE CONTAINER APPS · CANADACENTRAL</div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {[
                    { name: 'auth-service',     port: 8081, status: 'OK',       response: '{"service":"auth","status":"ok"}',     color: C.green },
                    { name: 'ai-service',        port: 8085, status: 'OK',       response: '{"service":"ai","status":"ok"}',       color: C.green },
                    { name: 'pipeline-service',  port: 8086, status: 'OK',       response: '{"service":"pipeline","status":"ok"}', color: C.green },
                    { name: 'features-service',  port: 8082, status: 'OK',       response: '{"service":"features","status":"ok"}', color: C.green },
                    { name: 'monitor-service',   port: 8083, status: 'DEGRADED', response: '{"status":"degraded"}',                color: C.amber },
                  ].map((s, i) => (
                    <div key={i} style={{
                      display: 'flex', alignItems: 'center', gap: 14,
                      background: C.panel2, border: `1px solid ${C.dimmer}`,
                      padding: '10px 16px', borderLeft: `2px solid ${s.color}`,
                    }}>
                      <div style={{ width: 7, height: 7, borderRadius: '50%', background: s.color, boxShadow: `0 0 8px ${s.color}`, flexShrink: 0 }} />
                      <span style={{ color: '#fff', fontWeight: 700, fontSize: 11, minWidth: 160 }}>{s.name}</span>
                      <span style={{ color: C.dim, fontSize: 9, minWidth: 60 }}>:{s.port}</span>
                      <span style={{ color: C.dim, fontSize: 9, flex: 1, fontFamily: 'monospace' }}>{s.response}</span>
                      <a href={`https://${s.name}.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health`}
                        target="_blank" rel="noreferrer"
                        style={{ color: s.color, fontSize: 9, textDecoration: 'none', letterSpacing: '0.08em' }}>
                        OPEN ↗
                      </a>
                      <span style={{ color: s.color, fontSize: 9, fontWeight: 700, minWidth: 60, textAlign: 'right' }}>{s.status}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Footer */}
          <div style={{ marginTop: 32, paddingTop: 16, borderTop: `1px solid ${C.dimmer}`, display: 'flex', justifyContent: 'space-between', fontSize: 9, color: C.dimmer, letterSpacing: '0.1em' }}>
            <span>FINTECH-SAAS-PIPELINE · charlesnet76 · Victoria BC Canada</span>
            <span>stg_transactions → fct_monthly_revenue · fct_revenue_by_category · fct_revenue_by_province · fct_status_summary</span>
          </div>
        </div>
      </div>
    </>
  );
}
