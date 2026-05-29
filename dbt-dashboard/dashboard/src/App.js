import React, { useState } from 'react';
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell, LineChart, Line, Legend
} from 'recharts';

const COLORS = ['#6366f1', '#22c55e', '#f59e0b', '#ef4444', '#3b82f6', '#ec4899', '#14b8a6', '#f97316', '#8b5cf6', '#06b6d4'];

// Static data mirroring dbt mart outputs — swap for API calls once a backend is wired
const revenueByCategory = [
  { category: 'transfers',      total_revenue: 312450.80, total_transactions: 512 },
  { category: 'travel',         total_revenue: 298320.50, total_transactions: 487 },
  { category: 'shopping',       total_revenue: 201540.20, total_transactions: 634 },
  { category: 'healthcare',     total_revenue: 156230.60, total_transactions: 498 },
  { category: 'utilities',      total_revenue: 98430.40,  total_transactions: 520 },
  { category: 'entertainment',  total_revenue: 76520.30,  total_transactions: 489 },
  { category: 'groceries',      total_revenue: 68940.70,  total_transactions: 521 },
  { category: 'food_and_drink', total_revenue: 42310.90,  total_transactions: 498 },
  { category: 'subscriptions',  total_revenue: 18650.20,  total_transactions: 512 },
  { category: 'transport',      total_revenue: 14320.80,  total_transactions: 498 },
];

const revenueByProvince = [
  { province: 'ON', total_revenue: 412340.50, total_transactions: 1580, unique_customers: 68 },
  { province: 'BC', total_revenue: 298430.20, total_transactions: 1142, unique_customers: 49 },
  { province: 'AB', total_revenue: 198320.80, total_transactions: 758,  unique_customers: 33 },
  { province: 'QC', total_revenue: 178450.30, total_transactions: 682,  unique_customers: 29 },
  { province: 'MB', total_revenue: 98320.60,  total_transactions: 378,  unique_customers: 16 },
  { province: 'SK', total_revenue: 78450.40,  total_transactions: 289,  unique_customers: 12 },
];

const monthlyRevenue = [
  { txn_month: '2025-01', total_revenue: 98430, total_transactions: 412 },
  { txn_month: '2025-02', total_revenue: 112340, total_transactions: 468 },
  { txn_month: '2025-03', total_revenue: 134560, total_transactions: 561 },
  { txn_month: '2025-04', total_revenue: 128940, total_transactions: 538 },
  { txn_month: '2025-05', total_revenue: 145320, total_transactions: 606 },
  { txn_month: '2025-06', total_revenue: 118760, total_transactions: 495 },
  { txn_month: '2025-07', total_revenue: 156430, total_transactions: 652 },
  { txn_month: '2025-08', total_revenue: 142560, total_transactions: 594 },
  { txn_month: '2025-09', total_revenue: 128340, total_transactions: 535 },
  { txn_month: '2025-10', total_revenue: 167430, total_transactions: 698 },
  { txn_month: '2025-11', total_revenue: 153280, total_transactions: 639 },
  { txn_month: '2025-12', total_revenue: 178920, total_transactions: 746 },
];

const statusSummary = [
  { status: 'completed', total_transactions: 4162, pct_of_total: 83.24 },
  { status: 'pending',   total_transactions: 625,  pct_of_total: 12.50 },
  { status: 'failed',    total_transactions: 213,  pct_of_total: 4.26  },
];

const fmt = (n) => `CAD $${Number(n).toLocaleString('en-CA', { minimumFractionDigits: 0 })}`;

const Card = ({ title, value, sub }) => (
  <div style={{ background: '#1e1e2e', borderRadius: 12, padding: '20px 24px', flex: 1, minWidth: 180 }}>
    <div style={{ color: '#888', fontSize: 13, marginBottom: 6 }}>{title}</div>
    <div style={{ color: '#fff', fontSize: 26, fontWeight: 700 }}>{value}</div>
    {sub && <div style={{ color: '#6366f1', fontSize: 12, marginTop: 4 }}>{sub}</div>}
  </div>
);

export default function App() {
  const [tab, setTab] = useState('overview');

  const totalRevenue = revenueByCategory.reduce((s, r) => s + r.total_revenue, 0);
  const totalTxns    = revenueByCategory.reduce((s, r) => s + r.total_transactions, 0);

  const tabs = ['overview', 'category', 'province', 'monthly'];

  return (
    <div style={{ background: '#13131f', minHeight: '100vh', color: '#fff', fontFamily: 'system-ui, sans-serif', padding: 32 }}>
      <div style={{ maxWidth: 1100, margin: '0 auto' }}>

        {/* Header */}
        <div style={{ marginBottom: 32 }}>
          <h1 style={{ margin: 0, fontSize: 28, fontWeight: 800, color: '#fff' }}>FinTech Data Pipeline</h1>
          <p style={{ margin: '6px 0 0', color: '#888', fontSize: 14 }}>Powered by dbt + PostgreSQL</p>
        </div>

        {/* KPI Cards */}
        <div style={{ display: 'flex', gap: 16, marginBottom: 32, flexWrap: 'wrap' }}>
          <Card title="Total Revenue (Completed)" value={fmt(totalRevenue)} sub="CAD 2025" />
          <Card title="Total Transactions"         value={totalTxns.toLocaleString()} sub="all statuses" />
          <Card title="Completion Rate"            value="83.2%" sub={`${statusSummary[0].total_transactions} completed`} />
          <Card title="Provinces Active"           value={revenueByProvince.length} sub="BC · ON · AB · QC · MB · SK" />
        </div>

        {/* Tabs */}
        <div style={{ display: 'flex', gap: 8, marginBottom: 24 }}>
          {tabs.map(t => (
            <button key={t} onClick={() => setTab(t)} style={{
              padding: '8px 18px', borderRadius: 8, border: 'none', cursor: 'pointer', fontSize: 14, fontWeight: 600,
              background: tab === t ? '#6366f1' : '#1e1e2e',
              color: tab === t ? '#fff' : '#888',
            }}>{t.charAt(0).toUpperCase() + t.slice(1)}</button>
          ))}
        </div>

        {/* Overview */}
        {tab === 'overview' && (
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24 }}>
            <div style={{ background: '#1e1e2e', borderRadius: 12, padding: 24 }}>
              <h3 style={{ marginTop: 0, color: '#ccc', fontSize: 14 }}>Transaction Status</h3>
              <ResponsiveContainer width="100%" height={240}>
                <PieChart>
                  <Pie data={statusSummary} dataKey="total_transactions" nameKey="status" cx="50%" cy="50%" outerRadius={90} label={({ status, pct_of_total }) => `${status} ${pct_of_total}%`}>
                    {statusSummary.map((_, i) => <Cell key={i} fill={COLORS[i]} />)}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div style={{ background: '#1e1e2e', borderRadius: 12, padding: 24 }}>
              <h3 style={{ marginTop: 0, color: '#ccc', fontSize: 14 }}>Revenue by Province</h3>
              <ResponsiveContainer width="100%" height={240}>
                <BarChart data={revenueByProvince} margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#2a2a3e" />
                  <XAxis dataKey="province" tick={{ fill: '#888', fontSize: 12 }} />
                  <YAxis tick={{ fill: '#888', fontSize: 11 }} tickFormatter={v => `$${(v/1000).toFixed(0)}k`} />
                  <Tooltip formatter={v => fmt(v)} />
                  <Bar dataKey="total_revenue" fill="#6366f1" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        )}

        {/* Category */}
        {tab === 'category' && (
          <div style={{ background: '#1e1e2e', borderRadius: 12, padding: 24 }}>
            <h3 style={{ marginTop: 0, color: '#ccc', fontSize: 14 }}>Revenue by Category (fct_revenue_by_category)</h3>
            <ResponsiveContainer width="100%" height={340}>
              <BarChart data={revenueByCategory} layout="vertical" margin={{ left: 20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a3e" />
                <XAxis type="number" tick={{ fill: '#888', fontSize: 11 }} tickFormatter={v => `$${(v/1000).toFixed(0)}k`} />
                <YAxis dataKey="category" type="category" tick={{ fill: '#ccc', fontSize: 12 }} width={120} />
                <Tooltip formatter={v => fmt(v)} />
                <Bar dataKey="total_revenue" fill="#22c55e" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Province */}
        {tab === 'province' && (
          <div style={{ background: '#1e1e2e', borderRadius: 12, padding: 24 }}>
            <h3 style={{ marginTop: 0, color: '#ccc', fontSize: 14 }}>Province Breakdown (fct_revenue_by_province)</h3>
            <ResponsiveContainer width="100%" height={320}>
              <BarChart data={revenueByProvince}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a3e" />
                <XAxis dataKey="province" tick={{ fill: '#888' }} />
                <YAxis tick={{ fill: '#888', fontSize: 11 }} tickFormatter={v => `$${(v/1000).toFixed(0)}k`} />
                <Tooltip formatter={v => fmt(v)} />
                <Legend />
                <Bar dataKey="total_revenue" name="Revenue" fill="#6366f1" radius={[4, 4, 0, 0]} />
                <Bar dataKey="total_transactions" name="Transactions" fill="#f59e0b" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Monthly */}
        {tab === 'monthly' && (
          <div style={{ background: '#1e1e2e', borderRadius: 12, padding: 24 }}>
            <h3 style={{ marginTop: 0, color: '#ccc', fontSize: 14 }}>Monthly Revenue Trend (fct_monthly_revenue)</h3>
            <ResponsiveContainer width="100%" height={320}>
              <LineChart data={monthlyRevenue}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a3e" />
                <XAxis dataKey="txn_month" tick={{ fill: '#888', fontSize: 11 }} />
                <YAxis tick={{ fill: '#888', fontSize: 11 }} tickFormatter={v => `$${(v/1000).toFixed(0)}k`} />
                <Tooltip formatter={v => fmt(v)} />
                <Legend />
                <Line type="monotone" dataKey="total_revenue" name="Revenue" stroke="#6366f1" strokeWidth={2} dot={{ r: 4 }} />
                <Line type="monotone" dataKey="total_transactions" name="Transactions" stroke="#22c55e" strokeWidth={2} dot={{ r: 4 }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}

        <div style={{ marginTop: 32, color: '#444', fontSize: 12, textAlign: 'center' }}>
          FinTech Data Pipeline · dbt models: stg_transactions → fct_revenue_by_category · fct_revenue_by_province · fct_monthly_revenue · fct_status_summary
        </div>
      </div>
    </div>
  );
}
