const https = require('https')
const redis = require('redis')
const publisher = redis.createClient({host: 'redis'})

const knex = require('knex')({
    client: 'pg',
    connection: {
        host: process.env.DB_HOST || "db",
        user: process.env.POSTGRES_USER,
        password: process.env.POSTGRES_PASSWORD,
        database: process.env.POSTGRES_DB
    }
})

knex.schema.createTableIfNotExists('gps', function (table) {
    table.increments('id')
    table.timestamp('timestamp')
    table.float('lat')
    table.float('lng')
    table.string('id_gps')
    table.float('speed')
    table.float('angle')
    table.integer('linea_id')
    table.string('interno')
}).catch(console.error)




let hash = ''

function gethash(fn) {
    https.get('https://www.gpsbahia.com.ar', (resp) => {
        let data = ''
        resp.on('data', chunk => {
            data += chunk
        })
        resp.on('end', () => {
            match = /hash2 = "(.*)"/g.exec(data)
            hash = match[1]
            console.log('newhash', hash)
        })

    }).on("error", (err) => {
        console.log("Error: " + err.message)
    })
    setTimeout(gethash, 60000)
}
gethash()

let baseurl = 'https://www.gpsbahia.com.ar/web/get_track_data'

function crawl() {
    [1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 30, 31].map(linea_id => `${baseurl}/${linea_id}/${hash}`).forEach(url => {
        https.get(url, (resp) => {
            let data = ''
            resp.on('data', chunk => {
                data += chunk
            })
            resp.on('end', () => {
                finalData = JSON.parse(data)
                if (finalData.hash) {
                    hash = finalData.hash
                }
                const parsed = JSON.parse(data)
                if (parsed.status === 'ok') {
                    parsed.data.forEach(d => {
                        knex.insert({
                            timestamp: d.dt_tracker,
                            lat: d.lat,
                            lng: d.lng,
                            angle: d.angle,
                            speed: d.speed,
                            id_gps: d.gps,
                            linea_id: d.linea_id,
                            interno: d.interno
                        }).into('gps').catch(console.error)
                        publisher.publish("update", "123")
                    })
                }
            })
        }).on("error", (err) => {
            console.log("Error: " + err.message)
        })
    })

    setTimeout(crawl, 5000)
}
setTimeout(crawl, 5000)