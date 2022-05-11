package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"eval/pkg/db"
	"eval/pkg/grpc/client"
	"eval/pkg/grpc/server"

	pbbuilder "eval/proto/builder"
	pbcache "eval/proto/cache"
	pbeval "eval/proto/engine"

	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	"gorm.io/gorm"
)

const (
	port = "0.0.0.0:50051"
)

type serverContext struct {
	log *zerolog.Logger
	v   *viper.Viper
	db  *gorm.DB
}

func buildImage(ctx context.Context, in *pbeval.BuildRequest) *pbeval.BuildResponse {
	// ok, a filure to connect here doesn' return error
	conn, err := client.Connect("eval-builder.eval.svc.cluster.local:50051")
	if err != nil {
		log.Fatalf("did not connect")
	}
	defer conn.Close()
	client := pbbuilder.NewBuilderServiceClient(conn)

	requester := pbbuilder.Requester{
		UserName: "USER",
		HostName: "HOSTNAME",
	}

	md, _ := metadata.FromIncomingContext(ctx)
	log.Printf("BUILD METADATA: %v", md["user"][0])

	ctx = metadata.NewOutgoingContext(ctx, md)

	response, err := client.Build(ctx, &pbbuilder.BuildRequest{
		Requester: &requester,
		CommitSHA: in.CommitSHA,
		Branch:    in.Branch,
		Target:    in.Target,
	})
	if err != nil {
		log.Printf("bad answer from builder")
	} else {
		log.Printf("response %v", response)
	}
	return &pbeval.BuildResponse{
		ImageName: response.ImageName,
		ImageTag:  response.ImageTag,
		Response:  response.Response,
	}
}

func get(evalID string) *pbcache.GetResponse {
	conn, err := client.Connect("eval-cache.eval.svc.cluster.local:50051")
	defer conn.Close()

	cache := pbcache.NewCacheServiceClient(conn)

	response, err := cache.Get(context.Background(), &pbcache.GetRequest{Evaluation: evalID})
	if err != nil {
		log.Fatalf("Error when calling Get: %s", err)
	}

	return response
}

type EvalInfo struct {
	//	gorm.Model
	ID uuid.UUID `gorm:"type:uuid;primary_key;"`

	User     string
	Hostname string

	State  string
	Future uuid.UUID `gorm:"type:uuid;"`
}

func (e *EvalInfo) BeforeCreate(tx *gorm.DB) error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	tx.Statement.SetColumn("ID", uuid)
	return nil
}

func (s *serverContext) Eval(ctx context.Context, in *pbeval.EvalRequest) (*pbeval.EvalResponse, error) {
	user := "unknown"
	hostname := "unknown"
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		user = md["user"][0]
		hostname = md["hostname"][0]
	}

	evalInfo := EvalInfo{
		User:     user,
		Hostname: hostname,
	}

	result := s.db.Create(&evalInfo)

	// why this condition?
	if result.Error != nil && result.RowsAffected != 1 {
		s.log.Err(result.Error).Msg("Error occurred while creating a new evaluation")
		return nil, result.Error
	}
	_ = get(evalInfo.ID.String())
	return &pbeval.EvalResponse{}, nil
}

func (s *serverContext) Build(ctx context.Context, in *pbeval.BuildRequest) (*pbeval.BuildResponse, error) {
	s.log.Info().Msg("Let's see this one")
	md, _ := metadata.FromIncomingContext(ctx)
	s.log.Info().Str("user", md["user"][0]).Msg("Metadata")
	response := buildImage(ctx, in)
	s.log.Info().Str("tag", response.ImageTag).Msg("response")
	return response, nil
	//	return &pbeval.BuildResponse{Response: "done"}, nil
}

func serviceRegister(server server.Server) func(*grpc.Server) {
	return func(s *grpc.Server) {
		context := serverContext{}
		context.log = server.Logger()
		context.v = server.Config()
		context.db, _ = db.NewDB("engine", &EvalInfo{})
		pbeval.RegisterEngineServiceServer(s, &context)
		reflection.Register(s)
	}
}

func echoString(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello")
}

func main() {
	go func() {
		http.HandleFunc("/", echoString)

		http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hi")
		})

		// baseDir := "/data/eval/certificates"
		// //		caCert      = "ca.crt"
		// clientCert := "tls.crt"
		// clientKey := "tls.key"

		log.Fatal(http.ListenAndServe(":8081", nil))

		// log.Fatal(http.ListenAndServeTLS(":8081",
		// 	baseDir+"/"+clientCert,
		// 	baseDir+"/"+clientKey,
		// 	nil))
	}()

	server := server.Build(port)
	server.RegisterService(serviceRegister(server))
	server.Start()
}

// type Product struct {
// 	gorm.Model
// 	Code  string
// 	Price uint
// }

// func playSQL() {

// 	db, err := gorm.Open(sqlite.Open("/data/sqlite/engine.db"), &gorm.Config{})
// 	if err != nil {
// 		panic("failed to connect database")
// 	}

// 	// Migrate the schema
// 	db.AutoMigrate(&Product{})

// 	// Create
// 	db.Create(&Product{Code: "D42", Price: 100})

// 	// Read
// 	var product Product
// 	db.First(&product, 1)                 // find product with integer primary key
// 	db.First(&product, "code = ?", "D42") // find product with code D42
// 	log.Printf("product %v", product)

// 	// Update - update product's price to 200
// 	db.Model(&product).Update("Price", 200)
// 	// Update - update multiple fields
// 	db.Model(&product).Updates(Product{Price: 200, Code: "F42"}) // non-zero fields
// 	db.Model(&product).Updates(map[string]interface{}{"Price": 200, "Code": "F42"})

// 	// Delete - delete product
// 	db.Delete(&product, 1)

// 	// os.Remove("/data/sqlite/engine.db")

// 	// db, err := sql.Open("sqlite3", "/data/sqlite/engine.db")
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	// defer db.Close()

// 	// sqlStmt := `
// 	// create table foo (id integer not null primary key, name text);
// 	// delete from foo;
// 	// `
// 	// _, err = db.Exec(sqlStmt)
// 	// if err != nil {
// 	// 	log.Printf("%q: %s\n", err, sqlStmt)
// 	// 	return
// 	// }

// 	// tx, err := db.Begin()
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	// stmt, err := tx.Prepare("insert into foo(id, name) values(?, ?)")
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	// defer stmt.Close()
// 	// for i := 0; i < 100; i++ {
// 	// 	_, err = stmt.Exec(i, fmt.Sprintf("こんにちわ世界%03d", i))
// 	// 	if err != nil {
// 	// 		log.Fatal(err)
// 	// 	}
// 	// }
// 	// tx.Commit()

// 	// rows, err := db.Query("select id, name from foo")
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	// defer rows.Close()
// 	// for rows.Next() {
// 	// 	var id int
// 	// 	var name string
// 	// 	err = rows.Scan(&id, &name)
// 	// 	if err != nil {
// 	// 		log.Fatal(err)
// 	// 	}
// 	// 	fmt.Println(id, name)
// 	// }
// 	// err = rows.Err()
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }

// 	// stmt, err = db.Prepare("select name from foo where id = ?")
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	// defer stmt.Close()
// 	// var name string
// 	// err = stmt.QueryRow("3").Scan(&name)
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	// fmt.Println(name)

// 	// _, err = db.Exec("delete from foo")
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }

// 	// _, err = db.Exec("insert into foo(id, name) values(1, 'foo'), (2, 'bar'), (3, 'baz')")
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }

// 	// rows, err = db.Query("select id, name from foo")
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// 	// defer rows.Close()
// 	// for rows.Next() {
// 	// 	var id int
// 	// 	var name string
// 	// 	err = rows.Scan(&id, &name)
// 	// 	if err != nil {
// 	// 		log.Fatal(err)
// 	// 	}
// 	// 	fmt.Println(id, name)
// 	// }
// 	// err = rows.Err()
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
// }
