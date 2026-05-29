with transactions as (
    select * from {{ ref('stg_transactions') }}
    where status = 'completed'
)

select
    category,
    count(*)                          as total_transactions,
    round(sum(amount)::numeric, 2)    as total_revenue,
    round(avg(amount)::numeric, 2)    as avg_transaction,
    round(min(amount)::numeric, 2)    as min_transaction,
    round(max(amount)::numeric, 2)    as max_transaction
from transactions
group by category
order by total_revenue desc
