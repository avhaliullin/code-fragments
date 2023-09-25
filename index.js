import pkg from 'ydb-sdk';

const {Driver, getLogger, getSACredentialsFromJson, IamAuthService, MetadataAuthService} = pkg;

let logger = getLogger();
let driver;

async function initDb() {
    logger.info('Driver initializing...');

    let authService;
    if (process.env.SA_KEY_FILE) {
        const saKeyFile = process.env.SA_KEY_FILE;
        const saCredentials = getSACredentialsFromJson('./' + saKeyFile);
        authService = new IamAuthService(saCredentials);
    } else {
        authService = new MetadataAuthService();
    }

    driver = new Driver({
        endpoint: process.env.ENDPOINT,
        database: process.env.DATABASE,
        authService,
    });
    const timeout = 10000;
    if (!(await driver.ready(timeout))) {
        logger.fatal(`Driver has not become ready in ${timeout}ms!`);
        process.exit(1);
    }
    logger.info('Driver ready');
}


async function registerConnection(connectionID) {
    const query = `
    UPSERT INTO connections (id)
    VALUES ('${connectionID}');
  `;

    await driver.tableClient.withSession(async (session) => {
        await session.executeQuery(query);
    });
}

async function unregisterConnection(connectionID) {
    const query = `
        DELETE
        FROM connections
        WHERE id == '${connectionID}';
    `;
    await driver.tableClient.withSession(async (session) => {
        return await session.executeQuery(query);
    });
}

async function listConnections() {
    const query = `SELECT id
                   FROM connections;`;
    const {resultSets} = await driver.tableClient.withSession(async (session) => {
        return await session.executeQuery(query);
    });
    let connections = [];
    let rs = resultSets[0];
    for (const row of rs.rows) {
        connections.push(row.items[0].textValue);
    }
    return connections;
}

let dbInitialized = false

export const handler = async function (event) {
    logger.info(event)
    if (!dbInitialized) {
        await initDb();
        dbInitialized = true;
    }
    let ctx = event["requestContext"];
    if (ctx["eventType"] === "CONNECT") {
        await registerConnection(ctx.connectionId);
        return {"statusCode": "200"};
    } else if (ctx["eventType"] === "DISCONNECT") {
        await unregisterConnection(ctx.connectionId);
        return {"statusCode": "200"};
    } else {
        return {"statusCode": "200"};
    }
}
