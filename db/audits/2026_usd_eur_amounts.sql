-- Audit nominal bill USD/EUR yang dibuat sebelum perbaikan serialisasi.
-- READ-ONLY: tidak mengubah data apa pun.
--
-- Konteks: frontend lama mengirim minor-int ("1050") untuk $10.50, sedangkan
-- backend money.FromString membacanya sebagai major unit -> tersimpan 100x.
-- Jalankan query ini untuk melihat dampaknya. Repair sengaja tidak disediakan
-- otomatis (fix forward); kalau perlu koreksi, buat migration terpisah setelah
-- hasil audit direview.

-- 1) Ringkasan jumlah bill & baris terkait per currency.
SELECT
  b.currency,
  COUNT(DISTINCT b.id) FILTER (WHERE b.id IS NOT NULL) AS bill_count,
  COUNT(DISTINCT e.id) AS expense_count,
  COUNT(DISTINCT ei.id) AS item_count,
  COUNT(DISTINCT p.id) AS payment_count
FROM bill b
LEFT JOIN expense e ON e.bill_id = b.id
LEFT JOIN expense_item ei ON ei.expense_id = e.id
LEFT JOIN payment p ON p.bill_id = b.id
WHERE b.currency IN ('USD', 'EUR')
GROUP BY b.currency;

-- 2) Detail expense (nilai sekarang vs nilai yang seharusnya /100).
SELECT
  b.public_id,
  b.currency,
  e.id AS expense_id,
  e.title,
  e.amount,
  e.amount / 100 AS amount_expected,
  e.tax,
  e.tax / 100 AS tax_expected,
  e.discount,
  e.discount / 100 AS discount_expected,
  e.charge,
  e.charge / 100 AS charge_expected
FROM expense e
JOIN bill b ON b.id = e.bill_id
WHERE b.currency IN ('USD', 'EUR')
ORDER BY b.id, e.id;

-- 3) Detail item.
SELECT
  b.public_id,
  b.currency,
  ei.id AS item_id,
  ei.name,
  ei.price,
  ei.price / 100 AS price_expected,
  ei.quantity
FROM expense_item ei
JOIN expense e ON e.id = ei.expense_id
JOIN bill b ON b.id = e.bill_id
WHERE b.currency IN ('USD', 'EUR')
ORDER BY b.id, e.id, ei.id;

-- 4) Detail payment.
SELECT
  b.public_id,
  b.currency,
  p.id AS payment_id,
  p.expense_id,
  p.paid_by,
  p.amount,
  p.amount / 100 AS amount_expected
FROM payment p
JOIN bill b ON b.id = p.bill_id
WHERE b.currency IN ('USD', 'EUR')
ORDER BY b.id, p.id;
