CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    userid INT NOT NULL,
    totalamount NUMERIC(10, 2) NOT NULL,
    orderdate TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orderitems (
    id SERIAL PRIMARY KEY,
    orderid INT NOT NULL,
    productid INT NOT NULL,
    FOREIGN KEY (orderid) REFERENCES orders(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_orders_userid ON orders(userid);
CREATE INDEX IF NOT EXISTS idx_orderitems_orderid ON orderitems(orderid);