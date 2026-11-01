const name = process.env.RS_NAME;
const hosts = process.env.RS_MEMBERS.split(",");
const configServer = process.env.RS_CONFIGSVR === "true";

const config = {
  _id: name,
  members: hosts.map((host, index) => ({
    _id: index,
    host: host + ":27017",
    priority: index === 0 ? 2 : 1,
  })),
};

if (configServer) {
  config.configsvr = true;
}

function initiated() {
  try {
    return rs.status().ok === 1;
  } catch (error) {
    return false;
  }
}

if (!initiated()) {
  printjson(rs.initiate(config));
}

while (!db.hello().primary) {
  sleep(500);
}

print(name + " ready, primary: " + db.hello().primary);
