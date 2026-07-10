app = proc do |env|
  [200, {'Content-Type' => 'text/plain'}, ['PASS']]
end

run app
