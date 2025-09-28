INSERT INTO symbols(code, name, market, sort_key) VALUES
('AAPL','Apple Inc.','NASDAQ',10),
('MSFT','Microsoft Corp.','NASDAQ',20),
('GOOGL','Alphabet Inc.','NASDAQ',30)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  market = VALUES(market),
  sort_key = VALUES(sort_key);