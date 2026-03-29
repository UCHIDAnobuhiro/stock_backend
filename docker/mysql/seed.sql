INSERT INTO symbols (code, name, market) VALUES
('NVDA', 'NVIDIA Corp', 'NASDAQ'),
('AAPL', 'Apple Inc.', 'NASDAQ'),
('GOOGL', 'Alphabet Inc. (Class A)', 'NASDAQ'),
('MSFT', 'Microsoft Corp.', 'NASDAQ'),
('AMZN', 'Amazon.com Inc.', 'NASDAQ'),
('AVGO', 'Broadcom Inc.', 'NASDAQ'),
('META', 'Meta Platforms, Inc.', 'NASDAQ'),
('TSLA', 'Tesla, Inc.', 'NASDAQ'),
('BRK.B', 'Berkshire Hathaway Inc.', 'NYSE'),
('LLY', 'Eli Lilly and Company', 'NYSE')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  market = VALUES(market);
