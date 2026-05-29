with transactions as (
    select * from {{ ref('stg_transactions') }}
)

select
    status,
    count(*)                          as total_transactions,
    round(sum(amount)::numeric, 2)    as total_amount,
    round(
        100.0 * count(*) / sum(count(*)) over ()
    , 2)                              as pct_of_total
from transactions
group by status
order by total_transactions desc
