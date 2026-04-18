-- write your UP migration here
CREATE TABLE "public"."chat_active" (
                                        "chat_id" int8 NOT NULL,
                                        "last_active_time" timestamp(6) NOT NULL,
                                        CONSTRAINT "chat_active_pkey" PRIMARY KEY ("chat_id")
)
;
