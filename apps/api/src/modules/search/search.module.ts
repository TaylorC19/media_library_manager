import { Module } from "@nestjs/common";
import { ProvidersModule } from "../providers/providers.module";
import { SearchController } from "./search.controller";

@Module({
  imports: [ProvidersModule],
  controllers: [SearchController]
})
export class SearchModule {}
