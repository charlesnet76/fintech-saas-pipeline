with transactions as (
    select * from {{ ref('stg_transactions') }}
    where status = 'completed'
)

select
    province,
    count(*)                          as total_transactions,
    round(sum(amount)::numeric, 2)    as total_revenue,
    round(avg(amount)::numeric, 2)    as avg_transaction,
    count(distinct customer_ref)      as unique_customers
from transactions
group by province
order by total_revenue desc
