with transactions as (
    select * from {{ ref('stg_transactions') }}
    where status = 'completed'
)

select
    txn_month,
    txn_year,
    count(*)                          as total_transactions,
    round(sum(amount)::numeric, 2)    as total_revenue,
    round(avg(amount)::numeric, 2)    as avg_transaction,
    count(distinct customer_ref)      as unique_customers,
    count(distinct category)          as categories_active
from transactions
group by txn_month, txn_year
order by txn_month
