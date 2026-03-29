INSERT INTO symbols(code, name, market) VALUES
('AAPL','Apple Inc.','NASDAQ'),
('MSFT','Microsoft Corp.','NASDAQ'),
('GOOGL','Alphabet Inc.','NASDAQ')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  market = VALUES(market);
