with source as (
    select * from {{ source('fintech', 'transactions') }}
),

cleaned as (
    select
        id                            as txn_pk,
        transaction_id,
        customer_ref,
        org_id,
        province,
        age_group,
        account_type,
        merchant,
        category,
        amount,
        currency,
        status,
        txn_date,
        txn_month,
        txn_year,
        loaded_at                     as _loaded_at
    from source
    where amount > 0
      and status in ('completed', 'pending', 'failed')
)

select * from cleaned
